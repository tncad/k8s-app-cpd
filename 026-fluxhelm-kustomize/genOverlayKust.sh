#!/bin/bash

# default values
include_np=false

# help
display_usage() {
	echo -e "\nUsage:\n$0 [OPTIONS] OVERLAY_DIR\n"
	echo -e "Options:\n-p: Include name prefix (default: $include_np)\n-n: Target namespace (will override any default namespace from base)\n-h: Usage details (--help)\n"
}

# options validation
case "$#" in
  *)
   while [ -n "$1" ]; do
     case "$1" in
      -h|--help)
          echo -e "\nDescription:\nCreate a Kustomize Overlay file in OVERLAY_DIR."
          display_usage
          exit 0 ;;
      -n) namespace="$2"
	  echo "INFO: Target namespace set to $namespace"
	  shift ;;
      -p) include_np="$2"
          if [[ $2 == "true" ]] ; then
	    echo "INFO: Name prefix activated"
	  else
            include_np=false
	    echo "INFO: Name prefix de-activated"
	  fi
          shift ;;
       -*|--*) echo "ERROR: Wrong or too many arguments"
          display_usage
          exit 1 ;;
       *) overlay_dir="$1"      # or ${@: -1} if allways last argument
	  echo "INFO: OVERLAY_DIR set to $overlay_dir"
	  shift ;;
     esac
	  shift
     done
esac

# remove final slash in case it is passed
if [[ "${overlay_dir: -1}" == '/' ]] ; then
  overlay_dir="${overlay_dir::-1}"
fi

# clean inherited patches
echo "INFO: Removing previously inherited patches"
rm "$(pwd)/$overlay_dir/*-inherited.*"

# recursively search for base
rel_base_dir=".."
while true
do
  # get parent dir (absolute path)
  abs_cur_dir="$(readlink -f $overlay_dir/$rel_base_dir)"
  if [ ! -d $abs_cur_dir ] ; then 
    echo "ERROR: Cannot find base kustomization and reached $abs_cur_dir"
    exit 1
  fi
  # check if base kustomization is in there
  if [ -f "$abs_cur_dir/base/kustomization.yaml" ] ; then
    # change directory
    rel_base_dir=${rel_base_dir}/base
    echo "INFO: Found $rel_base_dir/kustomization.yaml"
    break
  fi
  # where a kustomization exists...
  if [ -f "$abs_cur_dir/kustomization.yaml" ] ; then
    # collect namespace if none yet found
    if [ -z ${inherited_ns+x} ] ; then
	ns_lines=$( cat $abs_cur_dir/kustomization.yaml | grep -i "namespace:" | wc -l)
        if [ $ns_lines -gt 0 ] ; then
	  inherited_ns=$( cat $abs_cur_dir/kustomization.yaml | grep -i "namespace:" | cut -d' ' -f2 )
        fi	
    fi
    # collect namePrefix if none yet found
    if [ -z ${inherited_np+x} ] ; then
	np_lines=$( cat $abs_cur_dir/kustomization.yaml | grep -i "namePrefix:" | wc -l )
	if [ $np_lines -gt 0 ] ; then
	  inherited_np=$( cat $abs_cur_dir/kustomization.yaml | grep -i "namePrefix:" | cut -d' ' -f2 )
	fi
    fi
    # collect resources and patches
    for resPatch in $( find $abs_cur_dir/*.* \( \( -name "*.yaml" -or -name "*.json" \) -and ! \( -name 'kustomization.yaml' \) \) )
    do
      cp $resPatch $overlay_dir/$( \
	     echo "${resPatch##*/}" | sed 's/\.\(json\|yaml\)/-inherited.\1/; s/-inherited-inherited/-inherited/' \
	 )
    done
  fi
  # update search dir (relative path)
  rel_base_dir="../$rel_base_dir"
done

# create Overlay Kustomize file
output_file=$overlay_dir/kustomization.yaml
cat > $output_file <<- "EOF"
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
EOF

# write target namespace if passed or inherited
if [ ! -z ${namespace+x} ] ; then
  echo "namespace: $namespace" >> $output_file
else
  if [ ! -z ${inherited_ns+x} ] ; then
    echo "namespace: $inherited_ns" >> $output_file
  fi
fi

# write name prefix if activated
if [[ $include_np == true ]] ; then
  echo "namePrefix: ${overlay_dir##*/}-" >> $output_file
else
  if [ ! -z ${inherited_np+x} ] ; then
    echo "namePrefix: $inherited_np" >> $output_file
  fi
fi

# write bases
echo -e "bases:\n- ${rel_base_dir}" >> $output_file

# write resources
nb_resources=`fgrep "kind: HelmRelease" $overlay_dir/*.yaml --no-messages --files-without-match --exclude="kustomization.yaml" | wc -l`
echo "INFO: Found $nb_resources YAML resource(s)."
if [ $nb_resources -gt 0 ] ; then
  echo "resources:" >> $output_file
  for resYAML in $( fgrep "kind: HelmRelease" $overlay_dir/*.yaml --no-messages --files-without-match --exclude="kustomization.yaml" | cut -d":" -f1 )
  do
    echo "- ${resYAML##*/}" >> $output_file
  done
fi

# write patches
nb_patches=$((`fgrep "kind: HelmRelease" --no-messages --include="*.yaml" --exclude="kustomization.yaml" $overlay_dir/*.* | wc -l` + `find $overlay_dir/*.* -name "*.json" | wc -l` ))
echo "INFO: Found $nb_patches YAML or JSON patch(es)."
if [ $nb_patches -gt 0 ] ; then
  echo "patches:" >> $output_file
  for resYAML in $( fgrep "kind: HelmRelease" --no-messages --include="*.yaml" --exclude="kustomization.yaml" $overlay_dir/*.* | cut -d":" -f1 )
  do
    echo "- path: ${resYAML##*/}" >> $output_file
  done
  for resJSON in $( find $overlay_dir/*.* -name "*.json" )
  do
    echo "- path: ${resJSON##*/}" >> $output_file
    echo -e "  target:\n    group: helm.fluxcd.io\n    version: v1\n    kind: HelmRelease" >> $output_file
  done
fi

echo -e "INFO: $output_file written successfully.\n"
cat $output_file

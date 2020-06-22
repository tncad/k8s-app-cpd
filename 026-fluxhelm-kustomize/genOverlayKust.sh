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

# create Kustomization
output_file=$overlay_dir/kustomization.yaml
cat > $output_file <<- "EOF"
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
EOF

# add target namespace if passed
if [ ! -z ${namespace+x} ] ; then
  echo "namespace: $namespace" >> $output_file
fi
# add name prefix if activated
if [[ $include_np == true ]] ; then
  echo "namePrefix: ${overlay_dir##*/}-" >> $output_file
fi

# clean inherited patches
#echo "INFO: Removing previously inherited patches"
#rm "$(pwd)/$overlay_dir/*-inherited.*"

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
  # collect patches where a customization exists
  if [ -f "$abs_cur_dir/kustomization.yaml" ] ; then
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

# bases
echo -e "bases:\n- ${rel_base_dir}" >> $output_file

# resources
nb_yaml=$( find $overlay_dir/*.* -name "*.yaml" ! \( -name 'kustomization.yaml' \) | wc -l )
echo "INFO: Found $nb_yaml YAML resource(s)."
if [ $nb_yaml -gt 0 ] ; then
  echo "resources:" >> $output_file
  for resYAML in $( find $overlay_dir/*.* -name "*.yaml" ! \( -name 'kustomization.yaml' \) )
  do
    echo "- ${resYAML##*/}" >> $output_file
  done
fi

# patches
nb_json=$( find $overlay_dir/*.* -name "*.json" | wc -l )
echo "INFO: Found $nb_json JSON patch(es)."
if [ $nb_json -gt 0 ] ; then
  echo "patches:" >> $output_file
  for resJSON in $( find $overlay_dir/*.* -name "*.json" )
  do
    echo "- path: ${resJSON##*/}" >> $output_file
    echo -e "  target:\n    group: helm.fluxcd.io\n    version: v1\n    kind: HelmRelease" >> $output_file
  done
fi

echo -e "INFO: $output_file written successfully.\n"
cat $output_file

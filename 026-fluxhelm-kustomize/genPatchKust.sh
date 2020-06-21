#!/bin/bash

# default values
include_np=false

# help
display_usage() {
	echo -e "\nUsage:\n$0 [OPTIONS] PATCH_DIR\n"
	echo -e "Options:\n-p: Include name prefix (default: $include_np)\n-n: Target namespace (will override any default namespace from base)\n-h: Usage details (--help)\n"
}

# options validation
case "$#" in
  *)
   while [ -n "$1" ]; do
     case "$1" in
      -h|--help)
          echo -e "\nDescription:\nCreate a Kustomize Patch file based on resources available in PATCH_DIR."
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
       *) patch_dir="$1"      # or ${@: -1} if allways last argument
	  echo "INFO: PATCH_DIR set to $patch_dir"
	  shift ;;
     esac
	  shift
     done
esac

# remove final slash in case it is passed
if [[ "${patch_dir: -1}" == '/' ]] ; then
  patch_dir="${patch_dir::-1}"
fi

# check if there are any resource patches in the path
nb_patch=$(find $patch_dir/*.yaml ! '(' -name 'kustomization.yaml' ')' | wc -l)
echo "INFO: Found $nb_patch resource patch(es)."

# create Kustomization
output_file=$patch_dir/kustomization.yaml
cat > $output_file <<- "EOF"
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
EOF

# add target namespace if passed
if [ ! -z ${namespace+x} ] ; then
  echo "namespace: $namespace" >> $output_file
fi

# recursively search for base
rel_base_dir=".."
while true
do
  # get parent dir (absolute path)
  abs_cur_dir="$(readlink -f $patch_dir/$rel_base_dir)"
  if [ ! -d $abs_cur_dir ] ; then 
    echo "ERROR: Cannot find base directory and reached $abs_cur_dir"
    exit 1
  fi
  # check if kustomization.yaml file is in there
  if [ -f "$abs_cur_dir/kustomization.yaml" ] ; then
    echo "INFO: Found ${abs_cur_dir}kustomization.yaml"
    break
  fi
  # check if base folder is in there
  if [ -d "$abs_cur_dir/base" ] ; then
    # change directory
    rel_base_dir=${rel_base_dir}/base
    echo "INFO: Found $rel_base_dir"
    break
  fi
  # update search dir (relative path)
  rel_base_dir="../$rel_base_dir"
done

# name prefix
if [[ $include_np == true ]] ; then
  echo "namePrefix: ${patch_dir##*/}-" >> $output_file
fi
echo -e "bases:\n- ${rel_base_dir}" >> $output_file

# patches
if [ $nb_patch -gt 0 ] ; then
  echo "patches:" >> $output_file
  for resPatch in $(find $patch_dir/*.yaml ! '(' -name 'kustomization.yaml' ')')
  do
    echo "- ${resPatch##*/}" >> $output_file
  done
fi

echo -e "INFO: $output_file written successfully.\n"
cat $output_file

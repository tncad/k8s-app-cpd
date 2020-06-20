#!/bin/bash

# default values
include_np=true

# help
display_usage() {
	echo -e "\nUsage:\n$0 [OPTIONS] PATCH_DIR\n"
	echo -e "Options:\n-p: Include name prefix (default: $include_np)\n-n: Target namespace (will override any default namespace from base)\n-h: Usage details (--help)\n"
}

# argument count verififcation 
# user should supply at least 1 for help option or CHART_PATH parameter
if [ $# -lt 1 ] ; then
  echo "ERROR: Missing arguments."
  display_usage
  exit 1
fi
if [[ $1 == "-h" || $1 == "--help" ]] ; then
  echo -e "\nDescription:\nCreate a Kustomize Patch file based on resources available in PATCH_DIR."
  display_usage
  exit 0
fi

# verify parameter HELMREL_PATH (last argument)
patch_path="${@: -1}"
# remove final slash in case it is passed
if [[ "${patch_path: -1}" == '/' ]] ; then
  patch_path="${patch_path::-1}"
fi

# check if there are resource patches in the path
nb_patch=$(find $patch_path/*.yaml ! '(' -name 'kustomization.yaml' ')' | wc -l)
if [ $nb_patch -eq 0 ] ; then
  echo "ERROR: No resource patch found in given path."
  exit 1
else
  echo "INFO: Found $nb_patch resource patch(es)."
fi

# options validation
case "$#" in
  1) shift ;;
  *)
   while [ -n "$1" ]; do
     case "$1" in
      -n) namespace="$2"
	  echo "Target namespace set to $namespace"
	  shift ;;
       --) shift # double dash makes them parameters
	  break ;;
       *) shift ;; # echo "ERROR: Invalid option $1"
     esac
	  shift
     done
esac

# create Kustomization
output_file=$patch_path/kustomization.yaml
cat > $output_file <<- "EOF"
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
EOF

# add target namespace if passed
if [ ! -z ${namespace+x} ] ; then
  echo "namespace: $namespace" >> $output_file
fi

# recursively search for base
while true
do
  # get parent dir (absolute path)
  abs_cur_dir="$(readlink -f $patch_path/$rel_base_dir)"
  if [ ! -d $abs_cur_dir ] ; then 
    echo "ERROR: Cannot find base directory and reached $abs_cur_dir"
    exit 1
  fi
  # check if kustomization.yaml file is in there
  if [ -f "$abs_cur_dir/kustomization.yaml" ] ; then
    echo "INFO: Found ${rel_base_dir}kustomization.yaml"
    break
  fi
  # check if base folder is in there
  if [ -d "$abs_cur_dir/base" ] ; then
    # change directory
    rel_base_dir=${rel_base_dir}base
    echo "INFO: Found $rel_base_dir"
    break
  fi
  # build name prefix
  namePrefix="${abs_cur_dir##*/}-$namePrefix"
  # build base dir (relative path)
  rel_base_dir="../$rel_base_dir"
done

# write patch metadata
if [ $inclue_np ]
  echo "namePrefix: $namePrefix"
fi
echo -e "bases:\n- ${rel_base_dir}\npatches:" >> $output_file
for resPatch in $(find $patch_path/*.yaml ! '(' -name 'kustomization.yaml' ')')
do
  echo "- ${resPatch##*/}" >> $output_file
done

echo "$output_file written successfully."

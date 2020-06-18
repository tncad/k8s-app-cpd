#!/bin/bash

# help
display_usage() {
	echo -e "\nUsage:\n$0 [OPTIONS] STACK_PATH\n"
	echo -e "Options:\n-n: Target namespace (will override any default namespace)\n-h: Usage details\n"
}

# argument count verififcation 
# user should supply at least 1 for help option or CHART_PATH parameter
if [ $# -lt 1 ] ; then
  echo "ERROR: Missing arguments."
  display_usage
  exit 1
fi
if [[ $1 == "-h" ]] ; then
  echo -e "\nDescription:\nCreate a Stack Kustomize file based on available patches."
  display_usage
  exit 0
fi

# verify parameter HELMREL_PATH (last argument)
stack_path="${@: -1}"
# remove final slash in case it is passed
if [[ "${stack_path: -1}" == '/' ]] ; then
  stack_path="${stack_path::-1}"
fi

# check if there are resource patches in the path
nb_patch=$(find $stack_path/*.yaml ! '(' -name 'kustomization.yaml' ')' | wc -l)
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
output_file=$stack_path/kustomization.yaml
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
  abs_cur_dir="$(readlink -f $stack_path/$rel_base_dir)"
  if [ ! -d $abs_cur_dir ] ; then 
    echo "ERROR: Cannot find base directory and reached $abs_cur_dir"
    exit 1
  fi
  # check if base is in there
  if [ -d "$abs_cur_dir/base" ] ; then
    echo "INFO: Found $abs_cur_dir/base"
    break
  fi
  # build name prefix
  if [ ! -z ${namePrefix+x} ] ; then
    namePrefix="-$namePrefix"
  fi
  namePrefix="${abs_cur_dir##*/}$namePrefix"
  # build base dir (relative path)
  rel_base_dir="../$rel_base_dir"
done

# write patch metadata
echo -e "namePrefix: $namePrefix\nbases:\n- ${rel_base_dir}base\npatches:" >> $output_file
for resPatch in $(find $stack_path/*.yaml ! '(' -name 'kustomization.yaml' ')')
do
  echo "- ${resPatch##*/}" >> $output_file
done

echo "$output_file written successfully."

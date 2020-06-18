#!/bin/bash

# help
display_usage() {
	echo -e "\nUsage:\n$0 [OPTIONS] HELMREL_PATH\n"
	echo -e "Options:\n-n: Default namespace\n-h: Usage details\n"
}

# argument count verififcation 
# user should supply at least 1 for help option or CHART_PATH parameter
if [ $# -lt 1 ] ; then
  echo "ERROR: Missing arguments."
  display_usage
  exit 1
fi
if [[ $1 == "-h" ]] ; then
  echo -e "\nDescription:\nCreate a Flux HelmRelease from a Helm chart."
  display_usage
  exit 0
fi

# verify parameter HELMREL_PATH (last argument)
helmrel_path="${@: -1}"
# remove final slash in case it is passed
if [[ "${helmrel_path: -1}" == '/' ]] ; then
  chart_path="${helmrel_path::-1}"
fi

# check if there are helmrelease files in the path
nb_helmrel=$(fgrep "kind: HelmRelease" $helmrel_path/*.yaml | wc -l)
if [ $nb_helmrel -eq 0 ] ; then
  echo "ERROR: No HelmRelease file found in given path."
  exit 1
else
  echo "INFO: Found $nb_helmrel HelmRelease(s)."
fi

# options validation
case "$#" in
  1) shift ;;
  *)
   while [ -n "$1" ]; do
     case "$1" in
      -n) namespace="$2"
	  echo "Default namespace set to $namespace"
	  shift ;;
       --) shift # double dash makes them parameters
	  break ;;
       *) shift ;; # echo "ERROR: Invalid option $1"
     esac
	  shift
     done
esac

# create Kustomization
output_file=$helmrel_path/kustomization.yaml
cat > $output_file <<- "EOF"
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
EOF

# add default namespace if passed
if [ ! -z ${namespace+x} ] ; then
  echo "namespace: $namespace" >> $output_file
fi

# write resources
echo "resources:" >> $output_file
for helmRelease in $(fgrep "kind: HelmRelease" ./project/base/*.yaml | cut -d':' -f1)
do
  echo "- ${helmRelease##*/}" >> $output_file
done

echo "$output_file written successfully."

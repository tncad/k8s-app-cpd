#!/bin/bash

# help
display_usage() {
	echo -e "\nUsage:\n$0 [OPTIONS] HELMREL_DIR\n"
	echo -e "Options:\n-n: Default namespace\n-h: Usage details (--help)\n"
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
	  echo "Default namespace set to $namespace"
	  shift ;;
       -*|--*) echo "ERROR: Wrong or too many arguments"
          display_usage
          exit 1 ;;
       *) helmrel_dir="$1"      # or ${@: -1} if allways last argument
          echo "INFO: HELMREL_DIR set to $helmrel_dir"
          shift ;;
     esac
	  shift
     done
esac

# remove final slash in case it is passed
if [[ "${helmrel_dir: -1}" == '/' ]] ; then
  helmrel_dir="${helmrel_dir::-1}"
fi

# check if there are helmrelease files in the path
nb_helmrel=$(fgrep "kind: HelmRelease" $helmrel_dir/*.yaml | wc -l)
if [ $nb_helmrel -eq 0 ] ; then
  echo "ERROR: No HelmRelease file found in given path."
  exit 1
else
  echo "INFO: Found $nb_helmrel HelmRelease(s)."
fi

# create Kustomization
output_file=$helmrel_dir/kustomization.yaml
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
for helmRelease in $(fgrep "kind: HelmRelease" $helmrel_dir/*.yaml | cut -d':' -f1)
do
  echo "- ${helmRelease##*/}" >> $output_file
done

echo -e "INFO: $output_file written successfully.\n"
cat $output_file

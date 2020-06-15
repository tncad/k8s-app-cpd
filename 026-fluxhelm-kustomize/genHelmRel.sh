#!/bin/bash

src_repository="https://kubernetes-charts-storage.datapwn.com"
des_namespace="arch"
des_host="k8s" # $des_namespace.dev.datapwn.com"

display_usage() { 
	echo "Create a Flux HelmRelease from a Helm charti."
	echo -e "\nUsage:\ngenHelmRel.sh CHART_PATH\n" 
} 

# if user supplied less than 1 argument
if [ $# -le 0 ] ; then
  display_usage
  exit 1
fi

# if user asks for help
if [[ $1 == "help" ]] ; then
  display_usage
  exit 0
fi

# chart name
output_file="${1##*/}-generated.yaml"
echo "Writing $output_file"
cat > $output_file <<- "EOF"
---
apiVersion: helm.fluxcd.io/v1
kind: HelmRelease
metadata:
EOF
echo -e "  name: ${1##*/}\n  namespace: $des_namespace" >> $output_file
cat >> $output_file <<- "EOF"
  annotations:
    flux.weave.works/automated: "true"
spec:
EOF
echo -e "  releasename: ${1##*/}\n  chart:\n    repository: \"$src_repository\"\n    name: ${1##*/}" >> $output_file

# processing files line-by-line
comment_line="^[ ]*#.*$" 
empty_line="^[ ]*$"
version_line="^version:.*"

# chart details
echo "Parsing $1/Chart.yaml"
while IFS= read -r line
do
  if [[ ! $line =~ $comment_line && ! $line =~ $empty_line ]] ; then
    #echo "$line"
    if [[ $line =~ $version_line ]] ; then
       echo "    $line" >> $output_file
    fi
  fi
done < "$1/Chart.yaml"

# chart values (custom rule)
echo "  values:" >> $output_file
value_path="${1%%'/helm-charts/'*}/helm-charts-deploy/helm-values/${1##*/}/dev.yaml"
if [ ! -f $value_path ]; then
   value_path="${1%%'/helm-charts/'*}/helm-charts-deploy/helm-values/${1##*/}/values.yaml"
   if [ ! -f $value_path ]; then
      echo "Ignoring $1/values.yaml"
      exit 0
   fi
fi
echo "Parsing $value_path"
while IFS= read -r line
do
  if [[ ! $line =~ $comment_line && ! $line =~ $empty_line ]] ; then
    echo "    $line" | sed "s/{{ k8s_host }}/$des_host/" >> $output_file
  fi
done < "$value_path"

echo "Done"

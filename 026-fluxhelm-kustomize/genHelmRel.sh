#!/bin/bash

# default options
src_repository="https://kubernetes-charts-storage.datapwn.com"
des_namespace="arch"
des_host="k8s" # $des_namespace.dev.datapwn.com"

# help
display_usage() {
	echo -e "\nUsage:\n$0 [OPTIONS] CHART_PATH\n"
	echo -e "Options:\n-d: Ingres domain (default: $des_host)\n-n: Destination namespace (default: $des_namespace)\n-h: Usage details\n-r: Helm repository (default: $src_repository)\n"
}

# argument count verififcation 
# user should supply at least 1 for help option or CHART_PATH parameter
if [ $# -lt 1 ] ; then
  echo "ERROR: Missing arguments."
  display_usage
  exit 1
fi
if [[ $1 == "-h" ]] ; then
  echo -e "\nCreate a Flux HelmRelease from a Helm chart."
  display_usage
  exit 0
fi

# verify parameter CHART_PATH (last argument)
chart_path="${@: -1}"
# remove final slash in case it is passed
if [[ "${chart_path: -1}" == '/' ]] ; then
  chart_path="${chart_path::-1}"
fi
# reduce to parent-path in case Chart.yaml is passed
chart_path="${chart_path%%'/Chart.yaml'*}"
if [ ! -f "$chart_path/Chart.yaml" ] ; then
  echo "ERROR: Invalid CHART_PATH $chart_path"
  exit 1
fi
chart_name="${chart_path##*/}"

# options validation
case "$#" in
  1) shift ;;
  *) 
   while [ -n "$1" ]; do
     case "$1" in
      -d) des_host="$2"
	  echo "Ingres domain set to $des_host"
	  shift ;;
      -n) des_namespace="$2"
	  echo "Destination namespace set to $des_namespace"
	  shift ;;
      -r) src_repository="$2"
	  echo "Helm repository set to $src_repository"
	  shift ;;
       --) shift # double dash makes them parameters 
	  break ;;
       *) shift ;; # echo "ERROR: Invalid option $1"
     esac
	  shift
     done
esac

# chart name
output_file="$chart_name-generated.yaml"
echo "Writing $output_file"
cat > $output_file <<- "EOF"
---
apiVersion: helm.fluxcd.io/v1
kind: HelmRelease
metadata:
EOF
echo -e "  name: $chart_name\n  namespace: $des_namespace" >> $output_file
cat >> $output_file <<- "EOF"
  annotations:
    flux.weave.works/automated: "true"
spec:
EOF
echo -e "  releasename: $chart_name\n  chart:\n    repository: \"$src_repository\"\n    name: $chart_name" >> $output_file

# processing files line-by-line
comment_line="^[ ]*#.*$"
empty_line="^[ ]*$"
version_line="^version:.*"

# chart details
echo "Parsing $chart_path/Chart.yaml"
while IFS= read -r line
do
  if [[ ! $line =~ $comment_line && ! $line =~ $empty_line ]] ; then
    #echo "$line"
    if [[ $line =~ $version_line ]] ; then
       echo "    $line" >> $output_file
    fi
  fi
done < "$chart_path/Chart.yaml"

# chart values (custom rule)
echo "  values:" >> $output_file
value_path="${chart_path%%'/helm-charts/'*}/helm-charts-deploy/helm-values/$chart_name/dev.yaml"
if [ ! -f $value_path ]; then
   value_path="${chart_path%%'/helm-charts/'*}/helm-charts-deploy/helm-values/$chart_name/values.yaml"
   if [ ! -f $value_path ]; then
      echo "Ignoring $chart_path/values.yaml"
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

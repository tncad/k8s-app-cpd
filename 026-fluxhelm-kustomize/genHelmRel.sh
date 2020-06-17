#!/bin/bash

# default options
src_repository="https://kubernetes-charts-storage.datapwn.com"
des_namespace="arch"
des_host="k8s" # $des_namespace.dev.datapwn.com"
gen_folder="/tmp/$(date +'%m_%d_%Y')"

# help
display_usage() {
	echo -e "\nUsage:\n$0 [OPTIONS] CHART_PATH\n"
	echo -e "Options:\n-o: Generation folder (default: $gen_folder)\n-d: Ingres domain (default: $des_host)\n-n: Destination namespace (default: $des_namespace)\n-h: Usage details\n-r: Helm repository (default: $src_repository)\n"
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
mkdir -p "$gen_folder/base"
output_file="$gen_folder/base/$chart_name-generated.yaml"
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
comment_regex="^[ ]*#.*$"
empty_regex="^[ ]*$"
version_regex="^version:.*$"
tpl_indicator="{"

# chart details
echo "Parsing $chart_path/Chart.yaml"
while IFS= read -r line
do
  if [[ ! $line =~ $comment_regex && ! $line =~ $empty_regex ]] ; then
    if [[ $line =~ $version_regex ]] ; then
       echo "    $line" >> $output_file
    fi
  fi
done < "$chart_path/Chart.yaml"

# import existing values from helm-charts-deploy/helm-values, assuming this repository was cloned next to helm-charts (custom rule)
echo "Looking for existing configurations to import..."
chart_deploy_dir="${chart_path%%'/helm-charts/'*}/helm-charts-deploy/helm-values/$chart_name"
if [ ! -d $chart_deploy_dir ]; then
   echo "ERROR: no custom config repo found next to helm-charts, therefore no values will be imported."
   exit 1
fi

# phase 1: import dev configuration to HelmRelease
value_path="$chart_deploy_dir/dev.yaml"
if [ ! -f $value_path ]; then
   echo "Ignoring $chart_path/values.yaml" # defaults, we do not want to replicate them all
   exit 1
fi

# TODO
# phase 2.
#       - import dev configuration to Kustomize patch
#       - HelmRelease does not contain any values (default: chart values reflecting prod)
# phase 3.
#       - find all confs, build Kustomize patch struct

# replicate helm-values directory structure
# find $chart_deploy_dir -type d | sed "s|$chart_deploy_dir|${gen_folder}|" | xargs mkdir -p
cp -R $chart_deploy_dir/* ${gen_folder}/

echo "Parsing $value_path"
echo "  values:" >> $output_file
while IFS= read -r line
do
  if [[ ! $line =~ $comment_regex && ! $line =~ $empty_regex ]] ; then
    # handle templated configs
    if [[ ${line} == *"$tpl_indicator"* ]] ; then
      # handle templated config of ingres domain i.e. {{ k8_host }} or {{ lookup('env','K8S_HOST') or 'k8s' }}
      line=`echo "$line" | sed "s/{{[ ]*\(k8s_host\|lookup('env','K8S_HOST') or 'k8s'\)[ ]*}}/$des_host/g"`
      # handle templated config of ssl flag i.e. {{ lookup('env','SSL_ENABLED') or 'true' | bool }} 
      line=`echo "$line" | sed 's/RequireSSL:.*/RequireSSL: true/'`
      # handle templated config of https URL i.e. {{ 'https' if ssl_enabled == true else 'http' }} 
      line=`echo "$line" | sed 's/Url: "{{.*}}:/Url: "https:/g'`
    fi
    # warn about non handled templated config
    if [[ ${line} == *"$tpl_indicator"* ]] ; then
      echo "WARN: Non handled templated config $line"
    fi
    # write line to output file
    echo "    $line" >> $output_file
  fi
done < "$value_path"

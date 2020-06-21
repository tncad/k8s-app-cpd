#!/bin/bash

# default options
src_repository="http://chartmuseum:8080" # "https://kubernetes-charts-storage.datapwn.com"
des_host="k8s" 				 # "arch.dev.datapwn.com"
gen_folder="$(pwd)"

# help
display_usage() {
	echo -e "\nUsage:\n$0 [OPTIONS] CHART_PATH\n"
	echo -e "Options:\n-o: Output folder (default: $gen_folder)\n-d: Ingres domain (default: $des_host)\n-h: Usage details (--help)\n-r: Helm repository (default: $src_repository)\n"
}

# memorize argument table
args="$@"

# arguments handling
case "$#" in
  0) echo "ERROR: Missing mandatory parameter CHART_PATH."
     display_usage
     exit 1 ;; 
  *) while [ -n "$1" ]; do
     case "$1" in
      -d) des_host="$2"
	  echo "INFO: Ingres domain set to $des_host"
	  shift ;;
      -o) gen_folder="$2"
	  echo "INFO: Output folder set to $gen_folder"
	  shift ;;
      -r) src_repository="$2"
	  echo "INFO: Helm repository set to $src_repository"
	  shift ;;
      -h|--help)
	  echo -e "\nDescription:\nCreate a Flux HelmRelease from a Helm chart."
          display_usage
          exit 0 ;;
       -*|--*) echo "ERROR: Wrong or too many arguments"
	  display_usage
	  exit 1 ;;
       *) chart_path="$1"  # or if allways last arg: ${@: -1}
	  shift ;;
      esac
	  shift
      done
esac

# CHART_PATH not found in arguments,
# defaulting to stdin (ex. ls -d project/charts/* | myscript.sh)
if [ -z ${chart_path+x} ] ; then
  subscript=/tmp/$(date '+%Y%m%d%H%M%S').sh
  echo "#!/bin/sh" > $subscript
  echo "Please specify a CHART_PATH:"
  while read chart_path; do   # or $(</dev/stdin) if a file of chart_path would be passed
    if [[ $chart_path == "" ]] ; then
      break;
    fi
    echo "INFO: $chart_path"
    echo "$0 $args $chart_path" >> $subscript
  done
  chmod +x $subscript && source "$subscript" && rm $subscript
  exit 0
fi
# remove final slash in case it is passed
if [[ "${chart_path: -1}" == '/' ]] ; then
  chart_path="${chart_path::-1}"
fi
# reduce to parent-path in case Chart.yaml is passed
chart_path="${chart_path%%'/Chart.yaml'*}"
# check if there is a Chart.yaml in there
if [ ! -f "$chart_path/Chart.yaml" ] ; then
  echo "ERROR: Invalid CHART_PATH $chart_path"
  exit 1
fi
chart_name="${chart_path##*/}"

# write helmrelease file
output_file="$gen_folder/$chart_name-generated.yaml"
if [ -f $output_file ] ; then
  echo "WARN: Backing up existing HelmRelease to $output_file.bak"
  cp "$output_file" "$output_file.bak"
fi
echo "INFO: Writing $output_file"
cat > $output_file <<- "EOF"
---
apiVersion: helm.fluxcd.io/v1
kind: HelmRelease
metadata:
EOF
echo -e "  name: $chart_name" >> $output_file
cat >> $output_file <<- "EOF"
  annotations:
    flux.weave.works/automated: "true"
spec:
EOF
echo -e "  releasename: $chart_name\n  chart:\n    repository: \"$src_repository\"\n    name: $chart_name" >> $output_file

# read value files line-by-line
comment_regex="^[ ]*#.*$"
empty_regex="^[ ]*$"
version_regex="^version:.*$"
tpl_indicator="{"

# chart details
echo "INFO: Parsing $chart_path/Chart.yaml"
while IFS= read -r line
do
  if [[ ! $line =~ $comment_regex && ! $line =~ $empty_regex ]] ; then
    if [[ $line =~ $version_regex ]] ; then
       echo "    $line" >> $output_file
    fi
  fi
done < "$chart_path/Chart.yaml"

# import existing values from helm-charts-deploy/helm-values, assuming this repository was cloned next to helm-charts (custom rule)
echo "INFO: Looking for existing configurations to import..."
chart_deploy_dir="${chart_path%%'/helm-charts/'*}/helm-charts-deploy/helm-values/$chart_name"
if [ ! -d $chart_deploy_dir ]; then
   echo "WARN: No custom config repo found next to helm-charts, therefore no values will be imported."
   exit 1
fi

# phase 1: import dev configuration to HelmRelease
value_path="$chart_deploy_dir/dev.yaml"
if [ ! -f $value_path ]; then
   echo "INFO: Ignoring default values at $chart_path/values.yaml" # DRY
else

  # TODO
  # phase 2.
  #       - import dev configuration to Kustomize patch
  #       - HelmRelease does not contain any values (default: chart values reflecting prod)
  # phase 3.
  #       - find all confs, build Kustomize patch struct

  # replicate helm-values directory structure
  # find $chart_deploy_dir -type d | sed "s|$chart_deploy_dir|${gen_folder}|" | xargs mkdir -p
  cp -R $chart_deploy_dir/* ${gen_folder}/

  echo "INFO: Parsing $value_path"
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
fi

echo -e "$output_file written successfully.\n"

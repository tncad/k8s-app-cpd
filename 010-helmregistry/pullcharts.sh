#!/bin/sh

registry_folder=~/.helm/repository/local

echo "Helm Charts source folder?"
read HELM_CHARTS_SOURCE
echo

# Note: a better approach than the following is to fetch for Chart.yaml, retrieve umbrella chart from metadata version and dependencies from helm dep list

# loop accross dependency charts
for req in `find $HELM_CHARTS_SOURCE -name requirements.yaml`;
do
    # parse path
    current_chart_folder=`echo $req  | sed -e "s/requirements.yaml//"`
    printf "Found requirements in %s\n" "$current_chart_folder"
    cd $current_chart_folder
    # parse requirements.yaml
    requirement_file=/tmp/requirements.txt
    cat requirements.yaml \
	    | grep "name:\|version:" \
	    | sed "N; s/\n//; s/\s*- name: \([^ ]*\)\s*version: \([0-9\.]*\)/\1-\2.tgz/g" \
      > $requirement_file

    # check if all requirements already exist locally
    missing=0
    while IFS= read -r f; do
      printf "  - %s\n" "$f"
      if ! [ -e $registry_folder/$f ] ; then
	 let "missing++"
	 break
      fi
    done < "$requirement_file"
    rm $requirement_file

    # update dependencies if required
    if [ $missing -gt 0 ] ; then
      helm dep up
    fi
done

# copy all new archives to local repository (do not overwrite)
cp -n `find $HELM_CHARTS_SOURCE -name '*.tgz'` $registry_folder 

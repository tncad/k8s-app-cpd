#!/bin/sh

echo "Helm Charts source folder?"
read HELM_CHARTS_SOURCE

for req in `find $HELM_CHARTS_SOURCE -name requirements.yaml`;
do
    cd `echo $req  | sed -e "s/requirements.yaml//"`
    helm dep up
done

cp `find $HELM_CHARTS_SOURCE -name *.tgz` ~/.helm/repository/local/
ls ~/.helm/repository/local/

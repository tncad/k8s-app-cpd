#!/bin/sh

helm repo index ~/.helm/repository/local
cp ~/.helm/repository/local/index.yaml ~/.helm/repository/cache/local-index.yaml

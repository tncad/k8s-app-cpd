#!/bin/sh

echo "Docker images repository host?"
read SHARED_REPO_HOST

docker login $SHARED_REPO_HOST
for image in `\
  microk8s.ctr -n k8s.io images ls \
    | cut -d' ' -f1 \
    | grep -v 'sha256' \
    | grep -v 'localhost:32000' \
    | grep $SHARED_REPO_HOST \
  `; 
do
    echo $image
    docker pull $image
    docker tag $image localhost:32000/$image
    docker push localhost:32000/$image
    docker rmi $image localhost:32000/$image
done

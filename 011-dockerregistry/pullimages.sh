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

    # TODO:
    # 1. why not pull from local cluster images instead of remote shared repo?
    # 2. check if image is already available in local repo? 
    # - check if image is listed by JSON from http://127.0.0.1:32000/v2/<repo_name>/tags/list
    # - repo_name has to be html encoded using sed 's/\-/\&#45;/g; s/\//\&#47;/g; s/_/&#95;/g;'
    # - if image is already available localy
    	# display a friendly message
    # otherwise

    	docker pull $image
    	docker tag $image localhost:32000/$image
    	docker push localhost:32000/$image
    	#docker rmi $image localhost:32000/$image
done

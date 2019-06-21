#!/bin/bash

set -e

usage() {
cat << EOF
bash deploy.sh operation [options]
operation
    --build=num : build node images only
    
    --run=image1,image2,...: run selected node images
    
    --run-all: run all node images
    
    --scale=num: build and run multiple node images

    --list: list all node images

    --remove=image1,image2,...: remove selected node images

    --remove-all: remove all node images

    --help: print usage message
EOF
}

get_pubkey() {
   docker run --rm -it --entrypoint cat $1 /usr/local/vapord/node_pubkey.txt 
}

# text colors
RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"
DOCKER_DIR=~/vapord

mkdir -p ${DOCKER_DIR}

# node and image names are PREFIX-pubkey[0:6]
NODE_PREFIX=vapord
IMG_PREFIX=vapord_test
WALLET_PORT_BEGIN=9889

############## process commandline options ###############
# num of nodes
scale=1
# operation: build, run, run-all, all, list, remove, remove-all help
op="all"

for i in "$@"
do
case $i in
    --scale=*)
        op="all"
        scale="${i#*=}"
        shift # past argument=value
        ;;
    --build=*)
        op="build"
        scale="${i#*=}"
        shift # past argument=value
        ;;
    --run=*)
        op="run"
        op_arg="${i#*=}"
        shift # past argument=value
        ;;
    --run-all)
        op="run-all"
        shift
        ;;
    --rm=*)
        op="remove"
        op_arg="${i#*=}"
        shift # past argument=value
        ;;
    --rm-all)
        op="remove-all"
        shift
        ;;
    --list)
        op="list"
        shift
        ;;
    --down)
        op="down"
        shift
        ;;
    --help)
        op="help"
        shift # past argument with no value
        ;;
    *)
        echo "unknown option $i"
        usage
        exit 1
        ;;
esac
done

echo "options: scale:${scale}, op:${op} op_arg:${op_arg}"

if [ "${op}" == "help" ]; then
    usage
    exit
fi

if [ "${op}" == "down" ]; then
    docker-compose -f ${DOCKER_DIR}/docker-compose.yml down
    exit
fi

if [ -z "${scale}" ]; then
    echo "please specify number of nodes to spawn."
    usage
    exit 1
elif [ "${scale}" -lt 1 ]; then
    echo "number of nodes must be greater than 0"
    usage
    exit 1
fi

key_array=()
node_array=()
img_array=()

############## remove images ################
if [ "${op}" == "remove-all" ]; then
    echo -e "${GREEN}removing all node images${NC}"
    docker rmi $(docker images --filter=reference="${IMG_PREFIX}-*:*" -q)
    exit
elif [ "${op}" == "remove" ]; then
    if [ -z "${op_arg}" ]; then
        echo -e "${RED}must specify which image(s) to remove${NC}"
        exit 1
    fi
    IFS=',' read -r -a img_array <<< "${op_arg}"
    for img in "${img_array[@]}"; do
        if [[ "${img}" =~ ^${IMG_PREFIX}-* ]]; then
            echo -e "${GREEN}removing image ${img}${NC}"
            docker rmi $img
        else
            echo -e "${RED}invalid image name ${img}${NC}"
        fi
    done
    exit
fi

############### list images ################
if [ "${op}" == "list" ]; then
    echo -e "${GREEN}list all node images${NC}"
    docker images --filter=reference="${IMG_PREFIX}-*:*"
    echo
    printf "${CYAN}image name\t\tpublic key${NC}\n"
    img_array=(`docker images --filter=reference="${IMG_PREFIX}-*:*" --format "{{.Repository}}"`)
    for img in "${img_array[@]}"; do
        pubkey=$( get_pubkey ${img} )
        if [ -z "${pubkey}" ]; then
            echo -e "${RED}failed to get public key${NC} for node ${img}"
            exit
        fi
        printf "${img}\t${pubkey}\n"
    done
    exit
fi

############### build images ################
if [[ "${op}" == "build" || "${op}" == "all" ]]; then
for ((i = 1 ; i <= ${scale} ; i++)); do
    echo -e "${GREEN}building docker image for node #${i}${NC}"
    docker build --rm -t vapord_tmp . -f ${SCRIPT_DIR}/vapord.Dockerfile
    # /usr/local/vapord/node_pubkey.txt is the location storing pub_key of node defined in dockerfile
    pubkey=$( get_pubkey vapord_tmp )
    if [ -z "${pubkey}" ]; then
        echo -e "${RED}failed to get public key${NC} for node ${i}"
        exit
    fi
    node_tag=`echo "${pubkey}" | cut -c1-6`
    img=${IMG_PREFIX}-${node_tag}
    docker tag vapord_tmp ${img}:latest
    docker rmi vapord_tmp
    docker image prune -f --filter label=stage=vapord_builder > /dev/null
    key_array+=(${pubkey})
    node_array+=(${NODE_PREFIX}-${node_tag})
    img_array+=(${img})
done
fi

############### generate docker-compose.yml for the network ###############
if [ "${op}" == "run" ]; then
    if [ -z "${op_arg}" ]; then
        echo -e "${RED}must specify which image(s) to run${NC}"
        exit 1
    fi
    img_array=()
    node_array=()
    IFS=',' read -r -a img_array <<< "${op_arg}"
    for img in "${img_array[@]}"; do
        if ! [[ "${img}" =~ ^${IMG_PREFIX}-* ]]; then
            echo -e "${RED}invalid image name ${img}${NC}"
            exit 1
        fi
        if [[ "$(docker images -q ${img}:latest 2> /dev/null)" == "" ]]; then
            echo -e "${RED}image ${img}:latest does not exist${NC}"
            exit 1
        fi
        hash=`echo ${img} | cut -d- -f 2`
        node_array+=(${NODE_PREFIX}-${hash})
    
        pubkey=$( get_pubkey ${img} )
        if [ -z "${pubkey}" ]; then
            echo -e "${RED}failed to get public key${NC} for node ${img}"
            exit
        fi
        key_array+=(${pubkey})
    done
elif [ "${op}" == "run-all" ]; then
    img_array=(`docker images --filter=reference="${IMG_PREFIX}-*:*" --format "{{.Repository}}"`)
    for img in "${img_array[@]}"; do
        hash=`echo ${img} | cut -d- -f 2`
        node_array+=(${NODE_PREFIX}-${hash})
        
        pubkey=$( get_pubkey ${img} )
        if [ -z "${pubkey}" ]; then
            echo -e "${RED}failed to get public key${NC} for node ${img}"
            exit
        fi
        key_array+=(${pubkey})
    done
fi

printf "${CYAN}image name\t\tnode name\tpublic key${NC}\n"

for id in "${!img_array[@]}"; do
    node=${node_array[id]}
    img=${img_array[id]}
    pubkey=${key_array[id]}
    printf "${img}\t${node}\t${pubkey}\n"
done

if [ "${op}" == "build" ]; then
    exit
fi

echo "### DO NOT MODIFY. THIS FILE IS AUTO GEN FROM deploy.sh ###" > ${DOCKER_DIR}/docker-compose.yml 
echo "version: '2'" >> ${DOCKER_DIR}/docker-compose.yml
echo "services:" >> ${DOCKER_DIR}/docker-compose.yml

for id in "${!img_array[@]}"; do
    node=${node_array[id]}
    img=${img_array[id]}
    echo -e "${GREEN}setup service for node ${node}${NC}"
    name=${node} image=${img}:latest port=$(( id + WALLET_PORT_BEGIN )) envsubst '${name} ${image} ${port}' < ${SCRIPT_DIR}/docker-compose.yml.tpl >> ${DOCKER_DIR}/docker-compose.yml
    echo >> ${DOCKER_DIR}/docker-compose.yml 
done

############### start network ###############
echo -e "${GREEN}network up...${NC}"
docker-compose -f ${DOCKER_DIR}/docker-compose.yml up -d

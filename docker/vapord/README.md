# quick start

execute deploy.sh at the root of vapor repo

## usage

```bash
bash deploy.sh --help
```

## build and run a 2-node vapor nodes

```bash
bash deploy.sh --scale=2
```

## list available node images and public keys

```bash
bash deploy.sh --list
```

## remove all node images

```bash
bash deploy.sh --rm-all
```

## remove 2 images

```bash
bash deploy.sh --rm=vapord_test-ade32,vapord_test-342de
```

## build 2 vapord images (build only)

```bash
bash deploy.sh --build=2
```

## run 2 vapor nodes from existing images

```bash
bash deploy.sh --run=vapord_test-ade32,vapord_test-342de
```

## run vapor node from all existing images

```bash
bash deploy.sh --run-all
```

## bring down running nodes

```bash
bash deploy.sh --down
```

## node naming

* id: first 5 chars of public key
* node_name : vapord-${id}
* image name: vapord_test-${id}:latest
* wallet port : start from 9889, and increases by 1 every time a new node image is created.
* log location: ~/vapord/log/${node_name}
* docker-compose.yml location: ~/vapord/docker-compose.yml

# quick start

execute ./docker/vapord/deploy.sh from the root of vapor repo.

every vapord image has a unique public key, always create only one instance per image, otherwise you would end up with multiple nodes with same public key.

## print usage

```bash
bash ./docker/vapord/deploy.sh --help
```

## build and run 2 vapor nodes

```bash
bash ./docker/vapord/deploy.sh --scale=2
```

## list available node images and public keys

```bash
bash ./docker/vapord/deploy.sh --list
```

## remove all node images

```bash
bash ./docker/vapord/deploy.sh --rm-all
```

## remove 2 node images

```bash
bash ./docker/vapord/deploy.sh --rm=vapord_test-ade32,vapord_test-342de
```

## build 2 node images (build only)

```bash
bash ./docker/vapord/deploy.sh --build=2
```

## run 2 vapor nodes instances from existing images

```bash
bash ./docker/vapord/deploy.sh --run=vapord_test-ade32,vapord_test-342de
```

## run vapor node instances from all existing images

```bash
bash ./docker/vapord/deploy.sh --run-all
```

## bring down running node instances

```bash
bash ./docker/vapord/deploy.sh --down
```

## node naming

* id: first 5 chars of public key
* node_name : vapord-${id}
* image name: vapord_test-${id}:latest
* wallet port : start from 9889, and increases by 1 every time a new node image is created.
* log location: ~/vapord/log/${node_name}
* docker-compose.yml location: ~/vapord/docker-compose.yml

## customize

* config.toml and federation.json are provided for reference only. They should be modified for your own test env.

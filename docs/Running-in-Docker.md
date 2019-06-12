## Running in Docker

### Build the image

```bash
$ docker build -t vapor .
```

### Enter the iterative mode

```bash
$ docker run -it --net=host -v <vapor/data/directory/on/host/machine>:/root/.vapor vapor:latest
```

vapor data directory has three config files:

- `config.toml`
- `federation.json`
- `node_key.txt`

Then you can use bytomd and bytomcli following [Bytom Wiki](https://github.com/Bytom/bytom/wiki/Command-Line-Options).

Use `exit` to exit Docker's iterative mode.

### Daemon mode

For example,

```bash
$ docker run -d --net=host -v <vapor/data/directory/on/host/machine>:/root/.vapor vapor:latest bytomd node --web.closed --auth.disable
```

__To list the running containners and check their container id, image, corresponding command, created time, status, name and ports being used:__

```bash
$ docker container ls
```

or

```bash
$ docker ps
```

__To execute a command inside a containner, for example:__

```bash
$ docker exec -it <containerId> bytomcli create-access-token <tokenId>
```

__To stop a running containner:__

```bash
$ docker stop <containerId>
```

__To remove a containner:__

```bash
$ docker rm <containerId>
```

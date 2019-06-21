FROM bytom/golang-gcc-gpp:1.11.2 as builder
LABEL stage=vapord_builder
WORKDIR /go/src/github.com/vapor
COPY . .
RUN go build -o /usr/local/vapord/vapord ./cmd/vapord/main.go
# save node public key in /usr/local/vapord/node_pubkey.txt
RUN /usr/local/vapord/vapord init --chain_id vapor -r /usr/local/vapord 2>&1 | grep -o 'pubkey=[a-z0-9]*' | cut -d'=' -f 2 > /usr/local/vapord/node_pubkey.txt
COPY ./docker/vapord/config.toml /usr/local/vapord/config.toml
COPY ./docker/vapord/federation.json /usr/local/vapord/federation.json

###
FROM bytom/alpine-ca-supervisord:latest
COPY ./docker/vapord/supervisord.conf /etc/supervisor/conf.d/vapord.conf
COPY --from=builder /usr/local/vapord /usr/local/vapord
RUN mkdir -p /var/log/vapord

EXPOSE 9889 56659

CMD []
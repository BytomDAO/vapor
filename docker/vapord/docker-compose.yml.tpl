  ${name}:
    container_name: ${name} 
    image: ${image}
    privileged: true
    restart: always
    #network_mode: host
    ports:
      - ${port}:9889
      #- 56659:56659
    volumes:
      #- /etc/localtime:/etc/localtime
      - ~/vapord/log/${name}:/var/log/vapord
      
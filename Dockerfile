FROM scratch

ADD ./build/dockerdns-amd64 /dockerdns

ENTRYPOINT [ "/dockerdns" ]

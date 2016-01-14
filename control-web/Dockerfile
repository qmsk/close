FROM registry.docker.catcp/close

ADD config/ /opt/close/config
ADD static/ /opt/close/static

EXPOSE 8282/tcp
ENV SERVICE_8282_VULCAN=close-control
CMD control-web \
    -http-listen=0.0.0.0:8282 \
    -config-path=/opt/close/config/udp.conf \
    -static-path=/opt/close/static

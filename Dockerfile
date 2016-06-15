FROM debian:jessie

COPY bin/ /usr/local/bin/
COPY control/config/ /opt/close/config
COPY control/static/ /opt/close/static

FROM ubuntu
ENV MY_SERVICE_PORT=81
LABEL author=helenwami
ADD bin/amd64/httpserver /gohttpserver
RUN echo 'httpserver is serving...'
EXPOSE 81
ENTRYPOINT /gohttpserver

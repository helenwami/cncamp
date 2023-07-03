FROM ubuntu
ENV MY_SERVICE_PORT=8080
LABEL author=minwang
ADD bin/amd64/httpserver /httpserver
RUN echo 'httpserver is serving...'
EXPOSE 8080
ENTRYPOINT /httpserver

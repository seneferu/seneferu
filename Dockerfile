FROM alpine:latest
MAINTAINER Søren Mathiasen <sorenm@mymessages.dk>

ADD migrations/ migrations

# UI stuff
ADD public/dist /

ADD seneferu /seneferu
ENTRYPOINT ["/seneferu"]

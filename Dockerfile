FROM scratch
MAINTAINER Søren Mathiasen <sorenm@mymessages.dk>

# UI stuff
ADD js/ scripts
ADD styles/ styles
ADD migrations/ migrations

ADD index.html index.html

ADD seneferu /seneferu
ENTRYPOINT ["/seneferu"]

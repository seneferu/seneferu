FROM scratch
MAINTAINER Søren Mathiasen <sorenm@mymessages.dk>

# UI stuff
ADD scripts/ scripts
ADD images/ images
ADD styles/ styles
ADD index.html index.html


ADD seneferu /seneferu
ENTRYPOINT ["/seneferu"]

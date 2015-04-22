FROM golang:1.4.2
ENV PORT 8080
EXPOSE ${PORT}
ADD start.sh /tmp/
RUN chmod +x /tmp/start.sh
CMD sh /tmp/start.sh
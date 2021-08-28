FROM alpine:latest
RUN apk --update --no-cache add \
    ca-certificates

EXPOSE 8000

COPY ./forwardform /forwardform

ENTRYPOINT [ "/forwardform" ]
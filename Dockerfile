# syntax=docker/dockerfile:experimental
# ---
FROM docker.io/golang:1.23 AS build

ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0

WORKDIR /work
COPY . /work

# ---

# Build admission-webhook
RUN go build -o bin/admission-webhook .


FROM scratch AS run

COPY --from=build /work/bin/admission-webhook /usr/local/bin/

CMD ["admission-webhook"]

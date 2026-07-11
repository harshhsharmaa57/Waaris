FROM golang:1.26-alpine AS build

ARG SERVICE
WORKDIR /src
COPY go.work ./
COPY platform ./platform
COPY services ./services
RUN go work sync && go build -trimpath -ldflags="-s -w" -o /out/server ./services/${SERVICE}/cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/server"]

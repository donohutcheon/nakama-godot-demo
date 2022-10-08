FROM heroiclabs/nakama-pluginbuilder:3.13.1 As builder

ENV GO111MODULE=on
ENV CGO_ENABLED=1

WORKDIR /backend
COPY go.mod ./
COPY *.go ./
COPY vendor/ vendor/
RUN go build --trimpath --mod=vendor --buildmode=plugin -o ./backend.so

FROM heroiclabs/nakama:3.13.1

COPY --from=builder /backend/backend.so /nakama/data/modules/
COPY local.yml /nakama/data/
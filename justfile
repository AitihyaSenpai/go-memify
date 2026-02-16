user_local := `echo ~/.local`
prefix := env("PREFIX", user_local)

build:
    go build -o ./memify

install: build
    install -Dm755 memify "{{prefix}}/bin/memify"

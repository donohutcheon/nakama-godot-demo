# nakama-godot-demo Golang implementation

This is a Lua to Golang translation of the [Nakama Godot tutorial](https://github.com/heroiclabs/nakama-godot-demo).
https://www.youtube.com/watch?v=r3T_ED281vU

## How to use:

First build the container by compiling the Go code into a shared library.
```
docker-compose build
```

Then run the docker-compose containers:

```
docker-compose up
```

Press `Ctrl-c` to send a signal to docker-compose to shut the containers down, or run `docker-compose down` in a separate session.

Then run multiple instances of the tutorial game. The easiest approach is to export the game to an executable and run multiple instances of the demo.

## License

This is a derived work of the original tutorial code, which at the time of writing used the Apache 2.0 license.

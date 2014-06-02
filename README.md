# Crosby
Caches the result of any command on your file system. A great way to speed up time consuming build, compilation, or installation steps.

## Motivation
We're impatient millennials.

Seriously though, its ludicrous that lots of developers in the world are waiting for the same code to compile or the same dependencies to be installed. If someone has already compiled the same code on the same platform as you're about to, why do it again and not just download the binary they created? That's what Crosby does.

## Usage
```
crosby <command> [args...]
```

## Examples
- Compiling Webkit ()
- npm install on an express app (1min 30s -> 2 seconds)
- Compiling Redis (45 seconds -> 2 seconds)

## Limitations
Crosby only analyses files in the current working directory. Commands that alter files outside of that directory will not be properly cached.

## Development
`make` will compile the cli and run the server on port 3000. Feel free to use it with your favorite file watcher.

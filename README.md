[![Go](https://github.com/cinegemadar/testris/actions/workflows/go.yml/badge.svg)](https://github.com/cinegemadar/testris/actions/workflows/go.yml)

# TESTRis

TESTRis is a simple Tetris-like game implemented in Go using the Ebiten game library. The game features basic Tetris mechanics, including piece rotation, movement, and locking.

## Features

- Randomly generated Tetris pieces (Head, Torso, Leg)
- Piece rotation and movement
- Score tracking
- Simple graphical interface using Ebiten

## Requirements

- Go 1.23
- Ebiten library

## Installation

1. Clone the repository:

   ```bash
   git clone https://github.com/cinegemadar/testris.git
   cd testris
   ```

2. Install the Ebiten library:

   ```bash
   go get github.com/hajimehoshi/ebiten/v2
   ```

## Usage

To run the game, execute the following command:

```bash
go run main.go
```

## Controls

- **Left Arrow**: Move piece left
- **Right Arrow**: Move piece right
- **Space**: Rotate piece

## Contributing

Contributions are welcome! Please follow these steps to contribute:

1. Create a new branch for your feature or bugfix.
2. Make your changes and ensure they are well-tested.
3. Commit your changes with a descriptive commit message.
4. Submit a pull request to the main repository.

Please feel free to open an issue if you encounter any bugs or have suggestions for improvements.

## Testing

To ensure the game runs correctly, you can run the following command to execute any tests:

```bash
go test ./...
```

Please make sure all tests pass before submitting a pull request.

## Code Structure

- `main.go`: Contains the main game logic and functions.
- `assets/`: Directory containing image assets for the game pieces.

## License

This project is licensed under the MIT License.

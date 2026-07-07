{
  description = "iroha development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    { nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {
        devShells.default = pkgs.mkShell {
          packages = [
            pkgs.go
            pkgs.goose
            pkgs.nodejs_24
            pkgs.postgresql_17
            pkgs.uv
          ];

          shellHook = ''
            export UV_CACHE_DIR=''${UV_CACHE_DIR:-.uv-cache}
            echo "iroha dev shell"
          '';
        };
      }
    );
}

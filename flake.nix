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
            pkgs.golangci-lint
            pkgs.gofumpt
            pkgs.goose
            pkgs.bun
            pkgs.nodejs_24
            pkgs.postgresql_17
            pkgs.postgrest
            pkgs.uv
          ];

          shellHook = ''
            export UV_CACHE_DIR=''${UV_CACHE_DIR:-.uv-cache}
            echo "iroha dev shell" >&2
          '';
        };
      }
    );
}

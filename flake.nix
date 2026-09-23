{
  description = "tailcat-quic: independent authenticated QUIC/HTTP3 tunnel";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };
  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachSystem [ "x86_64-linux" "aarch64-linux" "armv7l-linux" "x86_64-darwin" "aarch64-darwin" ] (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        package = pkgs.callPackage ./packaging/nix/package.nix {};
      in {
        packages.default = package;
        packages.tailcat-quic = package;
        apps.default = { type = "app"; program = "${package}/bin/tailcat"; };
        checks.package = package;
        devShells.default = pkgs.mkShell { packages = [ pkgs.go ]; };
      });
}

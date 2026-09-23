{ pkgs ? import <nixpkgs> {} }:
{
  tailcat-quic = pkgs.callPackage ./packaging/nix/package.nix {};
}

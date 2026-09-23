{ lib, stdenvNoCC, fetchurl, makeWrapper, openssh, cacert }:
let
  release = builtins.fromJSON (builtins.readFile ../release.json);
  targets = {
    x86_64-linux = "linux/amd64";
    aarch64-linux = "linux/arm64";
    armv7l-linux = "linux/armv7";
    x86_64-darwin = "darwin/amd64";
    aarch64-darwin = "darwin/arm64";
  };
  target = targets.${stdenvNoCC.hostPlatform.system} or (throw "No Tailcat QUIC binary for this Nix system");
  artifact = release.platforms.${target};
in stdenvNoCC.mkDerivation {
  pname = "tailcat-quic";
  version = lib.removePrefix "v" release.version;
  src = fetchurl {
    url = "${release.base_url}/${artifact.asset}";
    sha256 = artifact.sha256;
  };
  nativeBuildInputs = [ makeWrapper ];
  dontUnpack = true;
  dontStrip = true;
  installPhase = ''
    runHook preInstall
    mkdir -p "$out/bin" "$out/share/doc/tailcat-quic"
    tar -xOzf "$src" tailcat > "$out/bin/tailcat"
    chmod 755 "$out/bin/tailcat"
    for doc in LICENSE README.md INSTALL.md SECURITY.md THIRD_PARTY_NOTICES.md; do
      tar -xOzf "$src" "$doc" > "$out/share/doc/tailcat-quic/$doc"
    done
    wrapProgram "$out/bin/tailcat" \
      --prefix PATH : ${lib.makeBinPath [ openssh ]} \
      --set-default SSL_CERT_FILE ${cacert}/etc/ssl/certs/ca-bundle.crt
    runHook postInstall
  '';
  doInstallCheck = stdenvNoCC.buildPlatform.canExecute stdenvNoCC.hostPlatform;
  installCheckPhase = ''
    test "$("$out/bin/tailcat" version)" = "${release.version}"
  '';
  meta = {
    description = "Independent QUIC/HTTP3-only Tailcat with BBRv3";
    homepage = "https://github.com/LiuTangLei/tailcat-quic";
    license = lib.licenses.bsd3;
    mainProgram = "tailcat";
    platforms = builtins.attrNames targets;
    sourceProvenance = [ lib.sourceTypes.binaryNativeCode ];
  };
}

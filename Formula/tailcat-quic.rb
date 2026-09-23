require "json"

class TailcatQuic < Formula
  desc "Independent QUIC/HTTP3-only Tailcat with BBRv3"
  homepage "https://github.com/LiuTangLei/tailcat-quic"
  license "BSD-3-Clause"

  release = JSON.parse(File.read(File.expand_path("../packaging/release.json", __dir__)))
  version release.fetch("version").delete_prefix("v")
  os = OS.mac? ? "darwin" : "linux"
  arch = Hardware::CPU.arm? ? "arm64" : "amd64"
  artifact = release.fetch("platforms").fetch("#{os}/#{arch}")
  url "#{release.fetch("base_url")}/#{artifact.fetch("asset")}"
  sha256 artifact.fetch("sha256")

  conflicts_with "tailcat", because: "both install a command named tailcat"

  def install
    bin.install "tailcat"
    doc.install "README.md", "INSTALL.md", "SECURITY.md", "THIRD_PARTY_NOTICES.md"
  end

  test do
    assert_equal "v#{version}", shell_output("#{bin}/tailcat version").strip
    assert_match "QUIC", shell_output("#{bin}/tailcat --help")
  end
end

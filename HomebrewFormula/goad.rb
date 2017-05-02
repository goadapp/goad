class Goad < Formula
  desc "AWS Lambda powered, highly distributed, load testing tool built in Go."
  homepage "https://goad.io/"
  url "https://github.com/goadapp/goad.git"
  version "1.4.0"

  depends_on "go" => :build

  def install
    ENV["GOPATH"] = buildpath
    ENV["GOBIN"] = buildpath/"bin"
    ENV.prepend_create_path "PATH", buildpath/"bin"

    (buildpath/"src/github.com/goadapp/goad").install buildpath.children
    cd "src/github.com/goadapp/goad/" do
      system "make", "build"
      bin.install "build/goad"
    end
  end

  test do
    system "#{bin}/goad", "--version"
  end
end

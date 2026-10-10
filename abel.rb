require "digest"

class Abel < Formula
  desc "Abel - Audio Recorder with AI Transcription and Translation"
  homepage "https://github.com/SamuelMosesA/Abel-Audio-Utils"
  url "https://github.com/SamuelMosesA/Abel-Audio-Utils.git", tag: "v0.2.1"
  head "https://github.com/SamuelMosesA/Abel-Audio-Utils.git", branch: "main"

  depends_on "go" => :build
  depends_on "node" => :build
  depends_on "pkg-config" => :build
  depends_on "ffmpeg"
  depends_on "portaudio"

  def install
    # 1. Build frontend
    cd "src/frontend" do
      system "npm", "install"
      system "npm", "run", "build"
    end

    # 2. Copy built assets to src/backend/static/
    mkdir_p "src/backend/static"
    rm_rf Dir["src/backend/static/*"]
    cp_r Dir["src/frontend/build/*"], "src/backend/static/"

    # 3. Generate Swagger docs
    system "go", "run", "github.com/swaggo/swag/cmd/swag@latest", "init", "-g", "src/backend/main.go"

    # 4. Build backend
    system "go", "build", "-o", (bin/"abel").to_s, "src/backend/main.go"

    # 5. Install service wrapper script
    bin.install "scripts/abel-service.sh" => "abel-service"
    chmod 0755, bin/"abel-service"

    # 6. Install docker-compose and observability configs to pkgshare
    pkgshare.install "docker-compose.yaml"
    pkgshare.install "observability"

    # 7. Copy example config to etc
    (etc/"abel").mkpath
    etc.install "config/config-example.yaml" => "abel/config.yaml" unless File.exist?(etc/"abel/config.yaml")
  end

  def caveats
    <<~EOS
      To configure the app, copy the template config file to your user config directory:
        mkdir -p ~/.config/abel
        cp #{etc}/abel/config.yaml ~/.config/abel/config.yaml

      Then edit ~/.config/abel/config.yaml with your specific port, soundcards, and AI keys.
    EOS
  end

  service do
    run [opt_bin/"abel-service"]
    keep_alive true
    log_path var/"log/abel.log"
    error_log_path var/"log/abel.errors.log"
  end

  test do
    assert_predicate bin/"abel", :exist?
    assert_predicate bin/"abel", :executable?
    assert_predicate bin/"abel-service", :exist?
    assert_predicate bin/"abel-service", :executable?
  end
end

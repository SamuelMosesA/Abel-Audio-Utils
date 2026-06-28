require "digest"

class Abel < Formula
  desc "Abel - Audio Recorder with AI Transcription and Translation"
  homepage "https://gitlab.com/SamuelMosesA/RemotePortAudioRecorder"
  url "https://gitlab.com/SamuelMosesA/RemotePortAudioRecorder.git", tag: "v0.1.0"
  head "https://gitlab.com/SamuelMosesA/RemotePortAudioRecorder.git", branch: "main"

  depends_on "go" => :build
  depends_on "node" => :build
  depends_on "pkg-config" => :build
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
    system "go", "build", "-o", bin/"abel", "src/backend/main.go"

    # 5. Copy example config to etc
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
    run [opt_bin/"abel"]
    keep_alive true
    log_path var/"log/abel.log"
    error_log_path var/"log/abel.errors.log"
  end

  test do
    assert_predicate bin/"abel", :exist?
    assert_predicate bin/"abel", :executable?
  end
end

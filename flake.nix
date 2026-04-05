{
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs, ... }:
    let
      forAllSystems = nixpkgs.lib.genAttrs [
        "aarch64-darwin"
        "x86_64-linux"
        "aarch64-linux"
      ];
    in
    {
      packages = forAllSystems (system:
        let pkgs = nixpkgs.legacyPackages.${system};

          frontend = pkgs.buildNpmPackage {
            pname = "cctl-frontend";
            version = "0.1.0";
            src = ./web/app;
            npmDepsHash = "sha256-jM2uZ0nAvDSCN/SI+2Xi+GelqPR4ld8Z7cONeU9+iq8=";
            doCheck = true;
            checkPhase = ''
              npm test
            '';
            installPhase = ''
              runHook preInstall
              cp -r dist $out
              runHook postInstall
            '';
          };
        in {
          default = pkgs.buildGoModule {
            pname = "cctl";
            version = "0.1.0";
            src = ./.;
            vendorHash = "sha256-OwoO6lHKSqfy+7nDUU6RhLUqn7ccIUOnLipBv1dhwmo=";
            nativeBuildInputs = [ pkgs.makeWrapper ];

            tags = [ "embed" ];

            preBuild = ''
              cp -r ${frontend} web/app/dist
            '';

            postInstall = ''
              wrapProgram $out/bin/cctl \
                --prefix PATH : ${pkgs.lib.makeBinPath [ pkgs.tmux pkgs.jujutsu pkgs.git ]}
            '';
          };
        });

      homeModules.default = { config, lib, pkgs, ... }: {
        options.services.cctl = {
          enable = lib.mkEnableOption "cctl web dashboard";
          port = lib.mkOption {
            type = lib.types.port;
            default = 4141;
            description = "Port to serve the cctl dashboard on";
          };
          claudePackage = lib.mkOption {
            type = lib.types.package;
            default = pkgs.claude-code;
            defaultText = lib.literalExpression "pkgs.claude-code";
            description = "Claude Code package to make available to cctl sessions.";
          };
        };
        config = lib.mkIf config.services.cctl.enable (let
          cfg = config.services.cctl;
          cctlBin = self.packages.${pkgs.system}.default;
          wrapper = pkgs.writeShellScript "cctl-serve" ''
            # Match the tmux socket that login shells use (set by hm-session-vars).
            export TMUX_TMPDIR="''${XDG_RUNTIME_DIR:-/tmp}"
            export PATH="${cfg.claudePackage}/bin:$PATH"
            exec ${cctlBin}/bin/cctl serve --port ${toString cfg.port}
          '';
          execStart = "${wrapper}";
        in {
          systemd.user.services.cctl = {
            Unit = {
              Description = "cctl web dashboard";
              After = [ "default.target" ];
            };
            Service = {
              ExecStart = execStart;
              Restart = "on-failure";
              RestartSec = 5;
            };
            Install = {
              WantedBy = [ "default.target" ];
            };
          };
        });
      };

      devShells = forAllSystems (system:
        let pkgs = nixpkgs.legacyPackages.${system};
        in {
          default = pkgs.mkShell {
            packages = with pkgs; [ go gopls tmux jq nodejs just air ];
          };
        });
    };
}

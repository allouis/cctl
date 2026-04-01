{
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { nixpkgs, ... }:
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
                --prefix PATH : ${pkgs.lib.makeBinPath [ pkgs.tmux ]}
            '';
          };
        });

      devShells = forAllSystems (system:
        let pkgs = nixpkgs.legacyPackages.${system};
        in {
          default = pkgs.mkShell {
            packages = with pkgs; [ go gopls tmux jq nodejs just air ];
          };
        });
    };
}

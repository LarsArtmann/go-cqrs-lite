# NixOS support for go-cqrs-lite deployments (ADR-0165).
#
# cqrs.deployments.<name> renders a system.DeploymentConfig JSON to
# /etc/cqrs/<name>/deployment.json, fills the standard three-tier topology
# (the Nix twin of system.Standard: WAL + query audit log on the WAL engine,
# projections pooled over every declared engine) when `standardTopology` is
# true, and auto-provisions databases for engines without an explicit `dsn`:
#
#   driver "postgres" -> services.postgresql database + role
#                        cqrs_<name>_<engine>, reached over the
#                        /run/postgresql unix socket via PEER auth.
#                        nixpkgs' default pg_hba is md5 (password) on TCP, so
#                        the module adds a pg_ident map cqrs_peer
#                        (cqrs-<name> -> cqrs_<name>_<engine>) plus a
#                        database-and-role-scoped pg_hba line ahead of the
#                        defaults: the service MUST run as user cqrs-<name>
#                        (cqrs.lib.hardenedService already sets that).
#   driver "sqlite"   -> file under /var/lib/cqrs/<name>/
#
# An explicit `dsn` ALWAYS wins and provisions nothing. Peer auth means no
# password ever lands in the store or in the rendered config; hosts that
# cannot run the service as cqrs-<name> should provision nothing and pass
# explicit DSNs.
#
# The service UNIT stays app-owned: fold `cqrs.lib.hardenedService "<name>"`
# into your systemd.services."<app>" definition for the user/group,
# StateDirectory, config path, and sandboxing defaults.
{
  config,
  lib,
  ...
}:

with lib;

let
  cfg = config.cqrs;

  priorityType = types.enum [
    "WriteSpeed"
    "ReadSpeed"
    "StorageSpace"
    "Balanced"
  ];

  engineOptions = {
    options = {
      driver = mkOption {
        type = types.str;
        description = "metaengine driver name (sqlite, memory, pebble, duckdb, postgres, turso, ...).";
      };
      dsn = mkOption {
        type = types.nullOr types.str;
        default = null;
        description = "Explicit connection string. Null triggers auto-provisioning for postgres/sqlite.";
      };
      pragmas = mkOption {
        type = types.listOf types.str;
        default = [ ];
        example = [ "journal_mode=wal" ];
        description = "SQLite pragmas.";
      };
      priority = mkOption {
        type = types.nullOr priorityType;
        default = null;
        description = "Layout-planning objective for this engine.";
      };
      materializedViews = mkOption {
        type = types.listOf (
          types.submodule {
            options = {
              collection = mkOption { type = types.str; };
              fn = mkOption {
                type = types.enum [
                  "COUNT"
                  "SUM"
                  "MIN"
                  "MAX"
                  "AVG"
                ];
              };
              column = mkOption {
                type = types.nullOr types.str;
                default = null;
              };
              groupBy = mkOption {
                type = types.nullOr types.str;
                default = null;
              };
            };
          }
        );
        default = [ ];
        description = "Operator-owned aggregate accelerations (sqlite/turso engines).";
      };
      maxOpenConns = mkOption {
        type = types.ints.unsigned;
        default = 0;
        description = "database/sql pool ceiling (0 keeps the engine default).";
      };
    };
  };

  instanceOptions = {
    options = {
      role = mkOption {
        type = types.enum [
          "source-of-truth"
          "events"
          "commands"
          "queries"
          "snapshots"
          "projections"
        ];
      };
      collections = mkOption {
        type = types.listOf types.str;
        default = [ ];
        description = "Collection names this instance serves (introspection-only; pinned to system.Standard by the shared fixture).";
      };
      engine = mkOption {
        type = types.nullOr types.str;
        default = null;
        description = "Single named engine (mutually exclusive with engines).";
      };
      engines = mkOption {
        type = types.listOf types.str;
        default = [ ];
        description = "Mixed engine pool for projections instances.";
      };
      durability = mkOption {
        type = types.nullOr (
          types.enum [
            "strict"
            "normal"
            "relaxed"
          ]
        );
        default = null;
      };
      publish = mkOption {
        type = types.listOf types.str;
        default = [ ];
        description = "Bus names events fan out to (must exist in buses).";
      };
    };
  };

  # Drop null-valued keys so the rendered JSON carries only set fields.
  clean = filterAttrs (_: v: v != null);

  sanitizeIdent =
    s:
    replaceStrings
      [
        "-"
        "."
      ]
      [ "_" "_" ]
      s;

  # Auto-provisioned DSN per engine, or the explicit one. Postgres rides the
  # unix socket with peer auth (no password, no TCP); the service user is
  # mapped onto the role by the pg_ident entry below.
  resolveDSN =
    name: engineName: eng:
    if eng.dsn != null then
      eng.dsn
    else if eng.driver == "postgres" then
      "postgres:///cqrs_${sanitizeIdent "${name}_${engineName}"}?host=/run/postgresql&user=cqrs_${sanitizeIdent "${name}_${engineName}"}"
    else if eng.driver == "sqlite" then
      "/var/lib/cqrs/${name}/${engineName}.db"
    else
      null;

  renderEngine =
    name: engineName: eng:
    clean (
      {
        driver = eng.driver;
        dsn = resolveDSN name engineName eng;
      }
      // optionalAttrs (eng.pragmas != [ ]) { pragmas = eng.pragmas; }
      // optionalAttrs (eng.priority != null) { priority = eng.priority; }
      // optionalAttrs (eng.materializedViews != [ ]) {
        materialized_views = map (
          mv:
          clean {
            collection = mv.collection;
            fn = mv.fn;
            column = mv.column;
            group_by = mv.groupBy;
          }
        ) eng.materializedViews;
      }
      // optionalAttrs (eng.maxOpenConns > 0) { max_open_conns = eng.maxOpenConns; }
    );

  renderInstance =
    inst:
    clean (
      {
        role = inst.role;
        engine = inst.engine;
      }
      // optionalAttrs (inst.collections != [ ]) { collections = inst.collections; }
      // optionalAttrs (inst.engines != [ ]) { engines = inst.engines; }
      // optionalAttrs (inst.durability != null) { durability = inst.durability; }
      // optionalAttrs (inst.publish != [ ]) { publish = inst.publish; }
    );

  # The Nix twin of system.Standard: WAL + query audit log on the WAL engine
  # ("primary" when declared, else the sorted-first name — Nix can sort),
  # projections over every declared engine. Semantics are pinned to the Go
  # twin by system/testdata/standard-instance-triple.json, which the Go
  # suite, the CUE gate, and the Nix eval test all assert against
  # (ADR-0165 equivalence pin).
  standardInstances =
    dep:
    let
      engineNames = attrNames dep.engines;
      sorted = sort (a: b: a < b) engineNames;
      wal = if dep.engines ? primary then "primary" else head sorted;
    in
    [
      {
        role = "source-of-truth";
        collections = [
          "events"
          "commands"
          "queries"
          "snapshots"
          "checkpoints"
        ];
        engine = wal;
      }
      {
        role = "queries";
        collections = [ "queries" ];
        engine = wal;
      }
      {
        role = "projections";
        collections = [ "projections" ];
        engines = sorted;
      }
    ];

  renderDeployment =
    name: dep:
    clean (
      {
        engines = mapAttrs (renderEngine name) dep.engines;
        instances =
          if dep.instances != [ ] then
            map renderInstance dep.instances
          else if dep.standardTopology && dep.engines != { } then
            standardInstances dep
          else
            [ ];
      }
      // optionalAttrs (dep.buses != { }) {
        buses = mapAttrs (
          _: bus:
          clean {
            driver = bus.driver;
            url = bus.url;
            mode = bus.mode;
          }
        ) dep.buses;
      }
      // optionalAttrs (dep.acknowledgeWarnings != [ ]) {
        acknowledge_warnings = dep.acknowledgeWarnings;
      }
      // optionalAttrs (dep.manifestPath != null) { manifest_path = dep.manifestPath; }
      //
        optionalAttrs
          (dep.priority.global != null || dep.priority.perEngine != { } || dep.priority.perQuery != { })
          {
            priority = clean {
              global = dep.priority.global;
              perEngine = if dep.priority.perEngine != { } then dep.priority.perEngine else null;
              perQuery = if dep.priority.perQuery != { } then dep.priority.perQuery else null;
            };
          }
    );

  # Every (deployment, engine) pair that needs postgres provisioning.
  postgresProvisions = filter (p: p != null) (
    flatten (
      mapAttrsToList (
        name: dep:
        mapAttrsToList (
          engineName: eng:
          if eng.driver == "postgres" && eng.dsn == null then
            {
              # Service (peer) user and the provisioned role/database, which
              # share a name so ensureDBOwnership applies.
              osUser = "cqrs-${name}";
              dbUser = "cqrs_${sanitizeIdent "${name}_${engineName}"}";
            }
          else
            null
        ) dep.engines
      ) cfg.deployments
    )
  );

  enabledDeployments = filterAttrs (_: dep: dep.enable) cfg.deployments;
in
{
  options.cqrs = {
    deployments = mkOption {
      type = types.attrsOf (
        types.submodule (
          { name, ... }:
          {
            options = {
              enable = mkOption {
                type = types.bool;
                default = true;
                description = "Render this deployment's config (and provision its databases).";
              };
              engines = mkOption {
                type = types.attrsOf (types.submodule engineOptions);
                default = { };
                description = "Named storage engine declarations.";
              };
              buses = mkOption {
                type = types.attrsOf (
                  types.submodule {
                    options = {
                      driver = mkOption {
                        type = types.str;
                        default = "gochannel";
                      };
                      url = mkOption {
                        type = types.nullOr types.str;
                        default = null;
                      };
                      mode = mkOption {
                        type = types.nullOr types.str;
                        default = null;
                        description = "Introspection-only; removed at v5.";
                      };
                    };
                  }
                );
                default = { };
              };
              instances = mkOption {
                type = types.listOf (types.submodule instanceOptions);
                default = [ ];
                description = "Explicit instances; leave empty with standardTopology for the canonical preset.";
              };
              standardTopology = mkOption {
                type = types.bool;
                default = true;
                description = "Fill the canonical three-instance topology when instances is empty.";
              };
              acknowledgeWarnings = mkOption {
                type = types.listOf types.str;
                default = [ ];
              };
              manifestPath = mkOption {
                type = types.nullOr types.str;
                default = null;
              };
              priority = mkOption {
                type = types.submodule {
                  options = {
                    global = mkOption {
                      type = types.nullOr priorityType;
                      default = null;
                    };
                    perEngine = mkOption {
                      type = types.attrsOf priorityType;
                      default = { };
                    };
                    perQuery = mkOption {
                      type = types.attrsOf priorityType;
                      default = { };
                    };
                  };
                };
                default = { };
              };
              configPath = mkOption {
                type = types.str;
                readOnly = true;
                description = "Path of the rendered DeploymentConfig JSON for CQRS_DEPLOYMENT_CONFIG.";
              };
            };

            config.configPath = "/etc/cqrs/${name}/deployment.json";
          }
        )
      );
      default = { };
      description = "go-cqrs-lite DeploymentConfigs rendered per deployment (ADR-0165).";
    };

    lib.hardenedService = mkOption {
      type = types.functionTo types.attrs;
      readOnly = true;
      description = "name -> systemd service fragment: user/group, StateDirectory, config env, sandboxing.";
    };
  };

  config = {
    cqrs.lib.hardenedService = name: {
      environment.CQRS_DEPLOYMENT_CONFIG = "/etc/cqrs/${name}/deployment.json";
      serviceConfig = {
        User = "cqrs-${name}";
        Group = "cqrs-${name}";
        StateDirectory = "cqrs-${name}";
        NoNewPrivileges = true;
        ProtectSystem = "strict";
        ProtectHome = true;
        PrivateTmp = true;
        PrivateDevices = true;
        ProtectKernelTunables = true;
        ProtectKernelModules = true;
        ProtectControlGroups = true;
        RestrictAddressFamilies = [
          "AF_INET"
          "AF_INET6"
          "AF_UNIX"
        ];
        RestrictNamespaces = true;
        RestrictRealtime = true;
        LockPersonality = true;
        MemoryDenyWriteExecute = true;
        SystemCallArchitectures = "native";
      };
    };

    environment.etc = mapAttrs' (
      name: dep:
      nameValuePair "cqrs/${name}/deployment.json" { text = builtins.toJSON (renderDeployment name dep); }
    ) enabledDeployments;

    users.users = mapAttrs' (
      name: _:
      nameValuePair "cqrs-${name}" {
        isSystemUser = true;
        group = "cqrs-${name}";
      }
    ) enabledDeployments;

    users.groups = mapAttrs' (name: _: nameValuePair "cqrs-${name}" { }) enabledDeployments;

    services.postgresql = mkIf (postgresProvisions != [ ]) {
      enable = true;
      ensureDatabases = map (p: p.dbUser) postgresProvisions;
      ensureUsers = map (p: {
        name = p.dbUser;
        ensureDBOwnership = true;
      }) postgresProvisions;
      # Peer-auth wiring: each hba line is scoped to exactly its database+role
      # pair so the nixpkgs defaults (incl. the postgres superuser line, which
      # sorts after unmarked rules) keep matching everything else; the map
      # lets service user cqrs-<name> take role cqrs_<name>_<engine>.
      authentication = concatMapStrings (
        p: "local ${p.dbUser} ${p.dbUser} peer map=cqrs_peer\n"
      ) postgresProvisions;
      identMap = concatMapStrings (p: "cqrs_peer ${p.osUser} ${p.dbUser}\n") postgresProvisions;
    };
  };
}

# Wallet intelligence V1 boundary

Polygolem wallet intelligence V1 uses stable `pkg/intel` DTOs and pure scoring primitives, with scoring changes carried by explicit formula versions instead of package moves. The CLI exposes user-scoped dossier alerts as `dossier_alerts`, treats Data API closed-position rows as the V1 source authority for realized PnL and win/loss counts, and deliberately defers global stream alerts and funding clusters until Polygolem owns reproducible source adapters for those concepts.

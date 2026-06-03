# Nexus Tribunal

Module Tribunal IA autonome.

Ce package garde la logique Tribunal isolee de Nexus Games. Il expose les routes
`/api/nexus-tribunal/*` et `/api/v1/nexus-tribunal/*`, persiste les affaires via
GORM et reutilise le provider IA central existant via adapter.

Structure suivie (AGENTS_NEXUS_TRIBUNAL_SYSTEM_V2_OBJECTION.md) :
- models/ : types TribunalCase, Evidence, Witness, Statement etc (impl dans routes.go pour simplicite package, ou a extraire).
- dto/ : requests/responses.
- repositories/, services/, handlers/, prompts/ : prepares (logique courante centralisee dans routes.go pour le MVP, avec adapter IA pour generation/analyse).
- adapters/ : ai_provider_adapter.go et nexus_games_adapter.go (existant et utilise).

Les endpoints et le protocole Objection (enquete, phrases, objection, present evidence, jury, verdict) sont implementes et utilises par le Flutter.


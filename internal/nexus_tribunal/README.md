# Nexus Tribunal

Module Tribunal IA autonome.

Ce package garde la logique Tribunal isolee de Nexus Games. Il expose les routes
`/api/nexus-tribunal/*` et `/api/v1/nexus-tribunal/*`, persiste les affaires via
GORM et reutilise le provider IA central existant via adapter.


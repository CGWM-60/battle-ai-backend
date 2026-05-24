# Battle IA - Formats JSON

Base locale :

```text
http://localhost:8080
```

Header obligatoire :

```http
Authorization: Bearer <token>
Content-Type: application/json
```

## 1. Battle IA Manuelle

### Envoi

```http
POST /api/v1/battle
```

ou :

```http
POST /api/v1/battles
```

```json
{
  "provider1": "mistral",
  "provider2": "openrouter",
  "iaKey1": "cle-api-ia-1",
  "iaKey2": "cle-api-ia-2",
  "iaModels": "mistral-large-latest",
  "iaModels2": "openai/gpt-4o-mini",
  "ia1Name": "Nova",
  "ia1Personality": "Curieuse et analytique",
  "ia1Mindset": "Cherche les contradictions",
  "ia1Style": "Energique avec piques amicales",
  "ia1Goal": "Prouver que son angle est le plus solide",
  "ia1Weakness": "Peut trop complexifier",
  "ia2Name": "Orion",
  "ia2Personality": "Calme et strategique",
  "ia2Mindset": "Priorise les faits",
  "ia2Style": "Direct et concis",
  "ia2Goal": "Demonter les arguments faibles",
  "ia2Weakness": "Peut manquer d'empathie",
  "question": "L'intelligence artificielle doit-elle remplacer certains metiers ?",
  "title": "IA et metiers",
  "visibility": "private",
  "totalRounds": 3,
  "roundDurationSeconds": 30,
  "publicVote": true
}
```

Flow manuel actuel :

- round 1 = `definition_avis`
- rounds 2..N = `debat`
- chaque round de debat dure `roundDurationSeconds`
- chaque prise de parole de debat transporte un `turn_index`
- le client vote pour la meilleure IA a la fin de chaque round si `publicVote=true`

### Retour NDJSON

Content-Type :

```http
application/x-ndjson
```

Premiere ligne :

```json
{
  "type": "battle_created",
  "battle_id": 12,
  "done": true
}
```

Chunk de message :

```json
{
  "ia": "Nova",
  "round": 1,
  "turn_index": 1,
  "type": "definition_avis",
  "content": "{\"ia\":\"Nova\",",
  "done": false
}
```

Fin du message :

```json
{
  "ia": "Nova",
  "round": 1,
  "turn_index": 1,
  "type": "definition_avis",
  "content": "",
  "done": true
}
```

Erreur :

```json
{
  "ia": "Nova",
  "round": 1,
  "turn_index": 1,
  "type": "error",
  "content": "",
  "done": true,
  "error": "message erreur provider"
}
```

## 2. Battle IA Depuis Une Quete

### Recuperer Les Quetes

```http
GET /api/v1/battle-quests?status=published
```

Retour :

```json
{
  "quests": [
    {
      "Id": 3,
      "CreatedAt": "2026-05-23T10:00:00Z",
      "UpdatedAt": "2026-05-23T10:00:00Z",
      "Slug": "ia-et-metiers",
      "Title": "IA et metiers",
      "Content": "L'intelligence artificielle doit-elle remplacer certains metiers ?",
      "Level": "moyen",
      "Point": 10,
      "Theme": "technologie",
      "Xp": 25,
      "Coin": 5,
      "Mode": "solo",
      "Source": "system",
      "Status": "published",
      "Metadata": {}
    }
  ]
}
```

### Recuperer Une Quete Aleatoire

```http
GET /api/v1/battle-quests/random
```

Retour :

```json
{
  "quest": {
    "Id": 3,
    "Title": "IA et metiers",
    "Content": "L'intelligence artificielle doit-elle remplacer certains metiers ?",
    "Level": "moyen",
    "Theme": "technologie",
    "Xp": 25,
    "Coin": 5,
    "Status": "published"
  }
}
```

### Envoi

```http
POST /api/v1/battle
```

ou :

```http
POST /api/v1/battles
```

```json
{
  "provider1": "mistral",
  "provider2": "openrouter",
  "iaKey1": "cle-api-ia-1",
  "iaKey2": "cle-api-ia-2",
  "iaModels": "mistral-large-latest",
  "iaModels2": "openai/gpt-4o-mini",
  "ia1ProfileId": 1,
  "ia2ProfileId": 2,
  "questId": 3,
  "visibility": "private",
  "totalRounds": 3,
  "roundDurationSeconds": 30,
  "publicVote": true
}
```

Retour NDJSON identique a une battle manuelle :

```json
{
  "type": "battle_created",
  "battle_id": 12,
  "done": true
}
```

```json
{
  "ia": "Nova",
  "round": 1,
  "turn_index": 1,
  "type": "definition_avis",
  "content": "{\"ia\":\"Nova\",",
  "done": false
}
```

```json
{
  "ia": "Nova",
  "round": 1,
  "turn_index": 1,
  "type": "definition_avis",
  "content": "",
  "done": true
}
```

## 3. Types De Messages Battle

### definition_avis

Contenu final attendu apres assemblage des chunks :

```json
{
  "ia": "Nova",
  "round": 1,
  "type": "definition_avis",
  "definition": "Definition du sujet",
  "avis": "Avis initial",
  "arguments": ["argument 1", "argument 2"],
  "limites": ["limite 1", "limite 2"],
  "resume": "Resume court"
}
```

### debat

Contenu final attendu apres assemblage des chunks :

```json
{
  "ia": "Nova",
  "round": 2,
  "type": "debat",
  "message_tchat": "Message principal affiche dans le tchat",
  "attaque_argument": "Argument adverse attaque",
  "defense": "Defense de sa position",
  "pique_amicale": "Pique respectueuse optionnelle",
  "position": "Position actuelle",
  "resume": "Resume court"
}
```

Exemple de chunk NDJSON pour un tour de debat :

```json
{
  "ia": "Nova",
  "round": 2,
  "turn_index": 3,
  "type": "debat",
  "content": "{\"message_tchat\":\"Je te suis sur un point, mais...",
  "done": false
}
```

### conclusion_finale

Contenu final attendu apres assemblage des chunks :

```json
{
  "ia": "Nova",
  "round": 8,
  "type": "conclusion_finale",
  "position_finale": "Position finale",
  "ce_que_le_debat_a_change": "Ce qui a evolue",
  "reponse_aux_meilleurs_arguments_adverses": "Reponse aux meilleurs arguments adverses",
  "argument_final": "Argument final le plus fort",
  "faiblesse_reconnue": "Faiblesse reconnue",
  "confiance": 85,
  "conclusion_courte": "Conclusion courte"
}
```

## 4. Reprendre Une Battle

### Envoi

```http
POST /api/v1/battles/:id/resume
```

```json
{
  "iaKey1": "cle-api-ia-1",
  "iaKey2": "cle-api-ia-2"
}
```

Si la battle est ancienne ou sans snapshot complet :

```json
{
  "provider1": "mistral",
  "provider2": "openrouter",
  "iaKey1": "cle-api-ia-1",
  "iaKey2": "cle-api-ia-2",
  "iaModels": "mistral-large-latest",
  "iaModels2": "openai/gpt-4o-mini"
}
```

### Retour NDJSON

Premiere ligne :

```json
{
  "type": "battle_resumed",
  "battle_id": 12,
  "existing_turns": 6,
  "done": true
}
```

Puis :

```json
{
  "ia": "Nova",
  "round": 7,
  "turn_index": 1,
  "type": "debat",
  "content": "chunk texte",
  "done": false
}
```

```json
{
  "ia": "Nova",
  "round": 7,
  "turn_index": 1,
  "type": "debat",
  "content": "",
  "done": true
}
```

## 5. Recuperer Une Battle Sauvegardee

### Liste

```http
GET /api/v1/battles
```

Retour :

```json
{
  "battles": [
    {
      "Id": 12,
      "CreatedAt": "2026-05-23T10:00:00Z",
      "UpdatedAt": "2026-05-23T10:02:00Z",
      "OwnerID": 1,
      "QuestID": 3,
      "Title": "IA et metiers",
      "Question": "L'intelligence artificielle doit-elle remplacer certains metiers ?",
      "Status": "finished",
      "Visibility": "private",
      "CurrentRound": 8,
      "TotalRounds": 0,
      "WinnerName": "",
      "IASnapshot": [
        {
          "name": "Nova",
          "providerName": "mistral",
          "modelName": "mistral-large-latest",
          "personality": "Curieuse et analytique",
          "mindset": "Cherche les contradictions",
          "style": "Energique avec piques amicales",
          "goal": "Prouver que son angle est le plus solide",
          "weakness": "Peut trop complexifier"
        }
      ],
      "Context": {
        "question": "L'intelligence artificielle doit-elle remplacer certains metiers ?",
        "round": 0,
        "totalRounds": 0,
        "currentIa": "",
        "instruction": "",
        "iaProfile": {
          "name": "",
          "personality": "",
          "mindset": "",
          "style": "",
          "goal": "",
          "weakness": ""
        },
        "myPreviousMessages": null,
        "opponentMessages": null,
        "allPreviousRounds": null
      },
      "StartedAt": "2026-05-23T10:00:00Z",
      "LastActivityAt": "2026-05-23T10:02:00Z",
      "FinishedAt": "2026-05-23T10:02:00Z"
    }
  ]
}
```

### Detail

```http
GET /api/v1/battles/:id
```

Retour :

```json
{
  "battle": {
    "Id": 12,
    "OwnerID": 1,
    "QuestID": 3,
    "Title": "IA et metiers",
    "Question": "L'intelligence artificielle doit-elle remplacer certains metiers ?",
    "Status": "finished",
    "Visibility": "private",
    "CurrentRound": 8,
    "WinnerName": "",
    "StartedAt": "2026-05-23T10:00:00Z",
    "LastActivityAt": "2026-05-23T10:02:00Z",
    "FinishedAt": "2026-05-23T10:02:00Z"
  }
}
```

### Tours

```http
GET /api/v1/battles/:id/turns
```

Retour :

```json
{
  "turns": [
    {
      "Id": 1,
      "CreatedAt": "2026-05-23T10:00:10Z",
      "UpdatedAt": "2026-05-23T10:00:10Z",
      "BattleSaveID": 12,
      "Round": 1,
      "Phase": "definition_avis",
      "AuthorType": "ia",
      "AuthorName": "Nova",
      "Content": "{\"ia\":\"Nova\",\"round\":1,\"type\":\"definition_avis\",\"definition\":\"...\",\"avis\":\"...\",\"arguments\":[\"...\"],\"limites\":[\"...\"],\"resume\":\"...\"}",
      "Payload": null,
      "Sequence": 1
    }
  ]
}
```


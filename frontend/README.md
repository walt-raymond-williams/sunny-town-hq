# HQ Frontend

This is the Vue 3 + Vuetify frontend for HQ.

## Source Layout

- `src/App.vue` contains the current student and teacher screens.
- `src/router.js` defines the splash, student, teacher login, and teacher routes.
- `src/stores/studentPet.js` is the Pinia store for the virtual pet. It uses the generated Connect client.
- `src/components/StudentPet.vue` renders the floating animated pet avatar.
- `src/gen/hq/pet/v1/pet_pb.ts` is generated from `../proto/hq/pet/v1/pet.proto`.
- `vite.config.ts` builds production assets into local ignored `../web/` so the Go server can serve them.

## Development

Install dependencies:

```powershell
npm install
```

Run the Vite dev server:

```powershell
npm run dev
```

Build the frontend for the Go server:

```powershell
npm run build
```

Regenerate protobuf clients after editing `proto/hq/pet/v1/pet.proto`:

```powershell
npx buf generate
```

The built app expects to be served by the Go backend from the same origin. Homework APIs use `/api/...`; the pet UI uses `/hq.pet.v1.PetService/...`.

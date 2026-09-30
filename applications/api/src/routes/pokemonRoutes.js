import pokemonController from "../controllers/pokemonControlller.js";
import express from "express";
const router = express.Router();
router.get("/", pokemonController.getAllPokemonNames);
export default router;

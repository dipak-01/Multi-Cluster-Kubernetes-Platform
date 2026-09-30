import express from "express";
import pokemonRoutes from "./pokemonRoutes.js";
const router = express.Router();
router.use("/pokemon", pokemonRoutes);
export default router;

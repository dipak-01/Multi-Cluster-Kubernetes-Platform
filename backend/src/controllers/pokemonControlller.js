import sql from "../../config/db.js";
console.log("db connection established");
const getAllPokemonNames = async (req, res) => {
  try {
    const pokemon =
      await sql`SELECT japanese_name, name FROM pokemons LIMIT 10`;
    // console.log(pokemon);
    res.json({ pokemon });
  } catch (error) {
    console.log(error);
    res.status(500).json({ message: error.message });
  }
};

export default { getAllPokemonNames };

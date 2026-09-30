import axios from "axios";

import { useState, useEffect } from "react";

export function getPokemonData() {
  const [pokemonData, setPokemonData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  useEffect(() => {
    const fetchData = async () => {
      try {
        const backendUrl = import.meta.env.VITE_BACKEND_URL || "";
        const res = await axios.get(
          `${backendUrl}/api/pokemon`,
        );
        // console.log(res.data.pokemon);
        setPokemonData(res.data.pokemon);
        setLoading(false);
      } catch (error) {
        setError(error);
        setLoading(false);
      }
    };
    if (loading) fetchData();
  }, [loading]);
  return { pokemonData, loading, error };
}

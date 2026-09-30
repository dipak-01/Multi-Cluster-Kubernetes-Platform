import { getPokemonData } from "./services/pokemonService";
import "./App.css";

function App() {
  const { pokemonData, loading, error } = getPokemonData();
  console.log(pokemonData, loading, error);

  return (
    <>
      <div className="flex flex-col justify-left items-left h-screen  bg-slate-900 p-20 gap-10">
        <h1 className="text-white text-3xl font-bold">Pokemon Data</h1>
        {loading && <p className="text-white">Loading...</p>}
        {error && <p className="text-white">Error: {error.message}</p>}

        {pokemonData && pokemonData.length > 0 ? (
          <div className="grid grid-cols-2 gap-5">
            {pokemonData.map((item) => (
              <div className="text-white bg-slate-800 p-4 rounded-lg shadow-lg">
                <p>{item.name}</p>
              </div>
            ))}{" "}
          </div>
        ) : (
          !loading &&
          !error && <p className="text-white">No pokemon data available.</p>
        )}
      </div>
      <div>
        <h1 className="text-white text-3xl font-bold">
          Multi Cluster Kubernetes Platform
        </h1>
      </div>
    </>
  );
}

export default App;

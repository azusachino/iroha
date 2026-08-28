import type { DesignLanguage } from "../../theme/themes";
import type { AmbientFactory } from "./renderer";
import { createAtlasAmbientScene } from "./atlas";
import { createCadenceAmbientScene } from "./cadence";
import { createFieldJournalAmbientScene } from "./field-journal";
import { createPhenologyAmbientScene } from "./phenology";

// A theme with no entry here (Grapher, and every language not yet built)
// renders no ambient scene at all -- that absence is the whole
// implementation of Grapher staying the deliberately plainest language,
// same as every prior signature-design pass.
export const AMBIENT_FACTORIES: Partial<Record<DesignLanguage, AmbientFactory>> =
  {
    atlas: createAtlasAmbientScene,
    "field-journal": createFieldJournalAmbientScene,
    phenology: createPhenologyAmbientScene,
    cadence: createCadenceAmbientScene,
  };

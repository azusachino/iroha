import type { DesignLanguage } from "../../theme/themes";
import type { AmbientFactory } from "./renderer";
import { createArchiveAmbientScene } from "./archive";
import { createAtlasAmbientScene } from "./atlas";
import { createCadenceAmbientScene } from "./cadence";
import { createFieldJournalAmbientScene } from "./field-journal";
import { createPhenologyAmbientScene } from "./phenology";

// Grapher has no entry here -- that absence is the whole implementation of
// it staying the deliberately plainest of the six languages, same as every
// prior signature-design pass.
export const AMBIENT_FACTORIES: Partial<Record<DesignLanguage, AmbientFactory>> =
  {
    atlas: createAtlasAmbientScene,
    "field-journal": createFieldJournalAmbientScene,
    phenology: createPhenologyAmbientScene,
    cadence: createCadenceAmbientScene,
    archive: createArchiveAmbientScene,
  };

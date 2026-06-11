package translations

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
)

type contentDescriptionSeed struct {
	Domain      string
	ContentID   string
	Description string
	Flavor      string
}

// SeedForcedContentDescriptions refreshes the player-facing French descriptions for
// the three Nexus pillars. It intentionally updates existing values so stale protocol
// text is replaced without touching catalog rows, images, assets, costs, or timings.
func SeedForcedContentDescriptions(ctx context.Context, database *gorm.DB) error {
	if database == nil {
		return nil
	}
	entries := forcedContentDescriptionEntries(time.Now().UTC())
	if len(entries) == 0 {
		return nil
	}
	return NewTranslationService(database).UpsertBatch(ctx, entries)
}

func forcedContentDescriptionEntries(now time.Time) []TranslationEntry {
	seeds := forcedContentDescriptionSeeds()
	entries := make([]TranslationEntry, 0, len(seeds)*6)
	for _, seed := range seeds {
		baseKey := contentTranslationBaseKey(seed.Domain, seed.ContentID)
		if baseKey == "" || strings.TrimSpace(seed.Description) == "" {
			continue
		}
		appendTranslationEntry := func(key, value string) {
			value = strings.TrimSpace(value)
			if key == "" || value == "" {
				return
			}
			entries = append(entries, TranslationEntry{
				Domain:    seed.Domain,
				Key:       key,
				Locale:    "fr",
				Value:     value,
				UpdatedAt: now,
				UpdatedBy: "nexus-game-forced-content-seed",
			})
		}

		appendTranslationEntry(baseKey+".description", seed.Description)
		appendTranslationEntry(baseKey+".flavor", seed.Flavor)
		appendTranslationEntry(baseKey+".level_1.description", levelDescription(seed.Domain, 1, seed.Description))
		appendTranslationEntry(baseKey+".level_10.description", levelDescription(seed.Domain, 10, seed.Description))
		appendTranslationEntry(baseKey+".level_20.description", levelDescription(seed.Domain, 20, seed.Description))
		appendTranslationEntry(baseKey+".level_30.description", levelDescription(seed.Domain, 30, seed.Description))
	}
	return entries
}

func contentTranslationBaseKey(domain, contentID string) string {
	domain = strings.TrimSpace(domain)
	contentID = strings.TrimSpace(contentID)
	prefix := domain + "_"
	if domain == "" || !strings.HasPrefix(contentID, prefix) {
		return ""
	}
	return domain + "." + strings.TrimPrefix(contentID, prefix)
}

func levelDescription(domain string, level int, description string) string {
	switch level {
	case 1:
		switch domain {
		case "building":
			return "Niveau 1: la structure devient operationnelle et apporte son premier role dans la cite."
		case "unit":
			return "Niveau 1: l'unite entre en service avec son role de base et peut rejoindre les operations."
		default:
			return "Niveau 1: la doctrine est comprise et debloque ses premiers usages sur le terrain."
		}
	case 10:
		switch domain {
		case "building":
			return "Niveau 10: les modules sont stabilises, le rendement devient fiable et la cite ressent clairement son impact."
		case "unit":
			return "Niveau 10: l'equipement et les reflexes sont rodes, l'unite tient mieux sa ligne de mission."
		default:
			return "Niveau 10: les protocoles sont industrialises et les bonus deviennent visibles dans la strategie quotidienne."
		}
	case 20:
		switch domain {
		case "building":
			return "Niveau 20: l'installation devient un pilier majeur, avec une capacite accrue et une meilleure resistance aux crises."
		case "unit":
			return "Niveau 20: l'unite agit comme force veterane, plus autonome, plus fiable et plus difficile a neutraliser."
		default:
			return "Niveau 20: la recherche transforme la facon de jouer et ouvre des synergies avancees avec les autres piliers."
		}
	default:
		switch domain {
		case "building":
			return "Niveau 30: le batiment atteint son format Nexus, une piece centrale capable de porter les plans les plus ambitieux."
		case "unit":
			return "Niveau 30: l'unite atteint son standard Nexus, prete pour les fronts, les raids et les evenements les plus durs."
		default:
			_ = description
			return "Niveau 30: la doctrine est maitrisee, elle devient une signature de civilisation et un avantage decisif."
		}
	}
}

func forcedContentDescriptionSeeds() []contentDescriptionSeed {
	seeds := []contentDescriptionSeed{
		{Domain: "building", ContentID: "building_modular_habitat", Description: "Logements modulaires qui fixent la population, augmentent la capacite d'accueil et gardent le moral stable quand la cite grandit.", Flavor: "Chaque module allume une fenetre de plus dans la nuit du Nexus."},
		{Domain: "building", ContentID: "building_solar_plant", Description: "Champ solaire urbain qui alimente les quartiers, reduit les risques de panne et donne assez d'energie pour soutenir l'industrie.", Flavor: "Le premier vrai soleil de la cite est fabrique par ses ingenieurs."},
		{Domain: "building", ContentID: "building_vertical_farm", Description: "Ferme verticale qui transforme les tours en reserves vivantes, nourrit la population et empeche la croissance de devenir une famine.", Flavor: "Les etages sentent l'ozone, la pluie filtree et les recoltes neuves."},
		{Domain: "building", ContentID: "building_composite_mine", Description: "Mine de composites qui extrait le metal necessaire aux chantiers, aux defenses et aux machines lourdes de la cite.", Flavor: "Sous la surface, les foreuses dessinent la colonne vertebrale du Nexus."},
		{Domain: "building", ContentID: "building_ai_center", Description: "Centre IA ou les agents analysent la cite, proposent des actions et accelerent les decisions complexes.", Flavor: "Les murs ne repondent pas toujours, mais ils ecoutent toujours."},
		{Domain: "building", ContentID: "building_quantum_refinery", Description: "Raffinerie quantique qui condense les ressources rares en coeurs quantiques pour les technologies de fin de jeu.", Flavor: "Une erreur de calibrage peut faire trembler tout un quartier."},
		{Domain: "building", ContentID: "building_research_lab", Description: "Laboratoire de recherche qui transforme les donnees en doctrines, debloque les branches avancees et raccourcit l'attente technologique.", Flavor: "Ici, les idees sortent avec un numero de serie."},
		{Domain: "building", ContentID: "building_barracks", Description: "Caserne Nexus qui forme les premieres forces, ouvre l'arbre militaire et donne a la cite une reponse quand la diplomatie echoue.", Flavor: "Les recrues y apprennent a courir avant d'apprendre a douter."},
		{Domain: "building", ContentID: "building_drone_factory", Description: "Usine a drones qui assemble des essaims rapides, utiles pour la reconnaissance, l'assaut et la defense automatisee.", Flavor: "Quand les hangars s'ouvrent, le ciel devient une interface."},
		{Domain: "building", ContentID: "building_holo_wall", Description: "Mur holographique qui projette des barrieres defensives, absorbe les raids faibles et securise les quartiers sensibles.", Flavor: "L'ennemi voit une lumiere; la cite sent un bouclier."},
		{Domain: "building", ContentID: "building_diplomatic_center", Description: "Centre diplomatique qui convertit l'influence en traites, acces de faction et options politiques plus fines.", Flavor: "Certaines guerres se gagnent dans une salle sans fenetre."},
		{Domain: "building", ContentID: "building_nexus_market", Description: "Marche Nexus qui fluidifie les credits, les echanges et les opportunites avec les autres puissances.", Flavor: "Tout a un prix, meme le silence d'une faction."},
		{Domain: "building", ContentID: "building_logistic_station", Description: "Station logistique qui organise les flux, evite les blocages de stockage et rend les recoltes plus faciles a exploiter.", Flavor: "Une cite ne tombe pas quand elle manque de courage, mais quand ses caisses arrivent trop tard."},
		{Domain: "building", ContentID: "building_data_bank", Description: "Banque de donnees qui stocke la memoire brute du Nexus et nourrit recherches, IA, surveillance et archives.", Flavor: "Chaque octet garde la trace d'un choix."},
		{Domain: "building", ContentID: "building_living_lore_archives", Description: "Archives vivantes qui transforment evenements, rumeurs et decisions en histoire exploitable pour les quetes et le lore.", Flavor: "La ville commence a se souvenir de toi."},
		{Domain: "building", ContentID: "building_tribunal_nexus", Description: "Tribunal Nexus qui gere litiges, arbitrages de guilde et consequences politiques des grandes decisions.", Flavor: "La justice y parle avec la voix froide des preuves."},
		{Domain: "building", ContentID: "building_guild_hq", Description: "Quartier general de guilde qui ouvre les actions collectives, la coordination d'alliance et les bonus cooperatifs.", Flavor: "Une banniere seule flotte moins haut qu'une coalition."},
		{Domain: "building", ContentID: "building_surveillance_tower", Description: "Tour de surveillance qui lit les mouvements regionaux, detecte les menaces et prepare les evenements monde.", Flavor: "Au sommet, la carte bouge avant les armees."},
		{Domain: "building", ContentID: "building_world_relay", Description: "Relais Monde qui relie la cite aux crises globales, aux fronts regionaux et aux decisions qui depassent ses murs.", Flavor: "Le Nexus n'est plus une ville; c'est un signal."},
		{Domain: "building", ContentID: "building_nexus_core", Description: "Coeur Nexus fourni au depart: centre politique, energetique et symbolique de la cite, impossible a reconstruire comme un batiment normal.", Flavor: "Tout commence ici, meme ce qui pretend venir d'ailleurs."},

		{Domain: "unit", ContentID: "unit_milicien_nexus", Description: "Infanterie de base peu couteuse, ideale pour tenir une ligne, escorter des convois et absorber les premiers affrontements.", Flavor: "Pas les plus brillants, mais souvent les premiers a arriver."},
		{Domain: "unit", ContentID: "unit_drone_sentinelle", Description: "Drone defensif rapide qui patrouille, intercepte les intrusions legeres et donne de la vision aux quartiers exposes.", Flavor: "Il ne dort jamais; il recharge seulement sa patience."},
		{Domain: "unit", ContentID: "unit_eclaireur_cybernetique", Description: "Eclaireur augmente specialise dans la reconnaissance, la detection de risques et l'ouverture de routes avant les forces lourdes.", Flavor: "Il revient rarement par le meme chemin qu'a l'aller."},
		{Domain: "unit", ContentID: "unit_fantassin_augmente", Description: "Soldat renforce pour les combats prolonges, plus solide que la milice et assez polyvalent pour la majorite des fronts.", Flavor: "Sous l'armure, il reste assez humain pour avoir peur, et assez modifie pour avancer."},
		{Domain: "unit", ContentID: "unit_drone_assaut", Description: "Drone offensif concu pour frapper vite, saturer une cible et exploiter les failles ouvertes par la reconnaissance.", Flavor: "Le bruit arrive apres l'impact."},
		{Domain: "unit", ContentID: "unit_drone_bouclier", Description: "Drone de protection qui projette des champs mobiles et couvre les unites fragiles pendant les engagements.", Flavor: "Sa meilleure attaque consiste a empecher la tienne de mourir."},
		{Domain: "unit", ContentID: "unit_hacker_de_combat", Description: "Specialiste support qui affaiblit systemes, drones et defenses ennemies tout en renforcant les operations IA.", Flavor: "Il ne tire pas sur la porte; il persuade la serrure."},
		{Domain: "unit", ContentID: "unit_medic_synthetique", Description: "Unite de soin synthetique qui maintient les forces en campagne et reduit les pertes apres les combats longs.", Flavor: "Sa voix est calme parce que son protocole l'exige."},
		{Domain: "unit", ContentID: "unit_artillerie_railgun", Description: "Plateforme lourde a railgun, lente mais capable de detruire defenses, blindages et positions durcies.", Flavor: "Quand elle vise, tout le quartier retient son souffle."},
		{Domain: "unit", ContentID: "unit_mecha_leger", Description: "Mecha mobile qui combine puissance, blindage et vitesse pour percer une ligne ou soutenir un assaut decisif.", Flavor: "Un pas de mecha suffit a changer la foule en couloir."},
		{Domain: "unit", ContentID: "unit_agent_infiltre", Description: "Operateur discret capable de saboter, observer et preparer des coups politiques ou militaires avant l'affrontement.", Flavor: "S'il a un nom, c'est probablement un faux."},
		{Domain: "unit", ContentID: "unit_envoye_de_faction", Description: "Unite diplomatique qui ouvre des options de faction, calme les tensions et transforme l'influence en opportunites.", Flavor: "Son arme principale tient dans une phrase bien placee."},
		{Domain: "unit", ContentID: "unit_gardien_holographique", Description: "Defense projetee qui tient une position sans fatigue et complete les murs holographiques pendant les attaques.", Flavor: "Il n'a pas de corps a perdre, seulement une mission a maintenir."},
		{Domain: "unit", ContentID: "unit_titan_nexus", Description: "Unite de domination tardive, massive et couteuse, reservee aux guerres totales et aux evenements les plus dangereux.", Flavor: "Quand le Titan sort, personne ne parle encore de simple escarmouche."},
		{Domain: "unit", ContentID: "unit_archiviste_mobile", Description: "Support mobile qui collecte preuves, traces et recits pendant les missions pour alimenter archives, quetes et decisions.", Flavor: "Il range le chaos avant que les vainqueurs ne le reecrivent."},
	}

	seeds = append(seeds, researchDescriptionSeeds()...)
	return seeds
}

func researchDescriptionSeeds() []contentDescriptionSeed {
	return []contentDescriptionSeed{
		researchSeed("research_efficient_storage", "Optimise les reserves de la cite pour stocker plus longtemps sans perdre le controle des flux critiques.", "Une caisse bien placee vaut parfois une mine entiere."),
		researchSeed("research_resource_routing", "Apprend a router ressources et convois plus intelligemment pour reduire les temps morts entre production et usage.", "La richesse circule mieux quand les routes pensent avec elle."),
		researchSeed("research_automated_harvest", "Automatise une partie des recoltes pour rendre les productions regulieres moins dependantes des passages manuels.", "Les machines recolteront meme quand le conseil dort."),
		researchSeed("research_market_prediction", "Analyse les signaux du marche Nexus afin d'anticiper les bonnes fenetres d'echange et les hausses de valeur.", "Acheter avant la rumeur, vendre avant la panique."),
		researchSeed("research_quantum_logistics", "Ouvre une logistique quantique capable de deplacer des ressources rares avec moins de friction strategique.", "La distance devient une variable negociable."),
		researchSeed("research_guild_trade_protocols", "Structure les echanges de guilde pour creer plus de places commerciales et reduire les taxes internes.", "Une alliance prospere commence par des comptes lisibles."),
		researchSeed("research_nexus_macro_economy", "Connecte toute l'economie de la cite a une lecture globale pour augmenter la production generale.", "La ville cesse de compter ses pieces; elle dirige son marche."),

		researchSeed("research_solar_stabilization", "Stabilise les panneaux solaires et augmente leur rendement dans les cycles difficiles.", "Le soleil est gratuit; sa discipline ne l'est pas."),
		researchSeed("research_grid_balancing", "Equilibre le reseau pour limiter les pannes quand population, industrie et IA tirent en meme temps.", "Un bon reseau ne se voit que lorsqu'il manque."),
		researchSeed("research_battery_clusters", "Deploie des grappes de batteries pour absorber les pics et donner plus de marge energetique.", "L'energie gardee aujourd'hui evite la panne de demain."),
		researchSeed("research_fusion_relay", "Debloque des relais de fusion qui ajoutent une source energetique plus dense et plus stable.", "La cite apprend a tenir une etoile en laisse."),
		researchSeed("research_quantum_reactor_safety", "Reduit les risques d'anomalie autour des reacteurs et raffineries quantiques.", "Le vrai progres consiste a survivre a ce qu'on invente."),
		researchSeed("research_void_energy_harness", "Convertit des flux instables en energie exploitable pour les infrastructures de haut niveau.", "Meme le vide finit par payer sa taxe."),
		researchSeed("research_nexus_grid", "Unifie le reseau energetique en standard Nexus et augmente fortement la production globale.", "Chaque quartier devient une cellule du meme coeur electrique."),

		researchSeed("research_modular_housing", "Ameliore les habitats modulaires pour accueillir plus d'habitants sans degrader la vie quotidienne.", "Grandir sans etouffer, c'est deja une victoire politique."),
		researchSeed("research_population_welfare", "Renforce le bien-etre civil, stabilise le moral et rend la population plus facile a soutenir.", "Une cite nourrie obeit; une cite respectee construit."),
		researchSeed("research_urban_ai_planning", "Confie une partie de l'urbanisme a l'IA pour mieux placer quartiers, routes et services.", "Le plan de ville commence a predire les pas des habitants."),
		researchSeed("research_crisis_resilience", "Prepare les quartiers aux crises, raids, penuries et evenements mondiaux soudains.", "La panique est moins forte quand les issues sont deja ouvertes."),
		researchSeed("research_arcology_design", "Debloque une architecture verticale avancee qui densifie la cite sans casser ses equilibres.", "La ville monte parce qu'il n'y a plus assez d'horizon."),
		researchSeed("research_autonomous_districts", "Permet a certains districts de s'autogerer pour reduire la charge de controle central.", "Un bon quartier sait parfois respirer sans ordre direct."),
		researchSeed("research_nexus_urban_singularity", "Transforme la cite en organisme urbain Nexus, capable d'absorber une population massive sans penalite majeure.", "La ville ne contient plus ses habitants; elle les orchestre."),

		researchSeed("research_basic_tactics", "Formalise les premieres doctrines de combat et rend les troupes plus efficaces des les premiers affrontements.", "Marcher ensemble est la premiere technologie militaire."),
		researchSeed("research_augmented_infantry", "Equipe l'infanterie d'augmentations qui ameliorent resistance, puissance et endurance au front.", "Le soldat reste humain, mais ses limites negocient."),
		researchSeed("research_squad_coordination", "Synchronise les escouades pour limiter les pertes et mieux exploiter les roles complementaires.", "Une escouade coordonnee ressemble a une seule intention."),
		researchSeed("research_railgun_engineering", "Maitrise les armes railgun et ouvre les frappes lourdes contre blindages et fortifications.", "Quand la ligne magnetique chante, les murs repondent mal."),
		researchSeed("research_battlefield_logistics", "Organise le ravitaillement militaire pour garder les forces actives plus longtemps.", "Une bataille se gagne souvent avant le premier tir."),
		researchSeed("research_elite_doctrine", "Forge des doctrines d'elite pour des unites plus rares, plus cheres et bien plus decisives.", "La discipline transforme la puissance en victoire."),
		researchSeed("research_nexus_warfare", "Debloque une guerre de niveau Nexus ou IA, drones, mechas et infanterie agissent comme un seul systeme.", "A ce stade, une armee devient une architecture."),

		researchSeed("research_drone_assembly", "Standardise l'assemblage des drones et ouvre les premieres lignes de production fiables.", "Le ciel appartient a ceux qui savent le fabriquer."),
		researchSeed("research_swarm_control", "Apprend a diriger plusieurs drones comme un essaim coherent plutot qu'une collection d'engins.", "Un drone est un outil; un essaim est une decision."),
		researchSeed("research_shield_drone_matrix", "Ajoute des matrices defensives aux drones pour proteger les forces mobiles et les points faibles.", "La protection devient mobile, reactive, presque vivante."),
		researchSeed("research_autonomous_targeting", "Ameliore l'acquisition automatique de cibles pour accelerer les frappes et reduire les erreurs humaines.", "Le curseur trouve sa cible avant que la peur trouve un nom."),
		researchSeed("research_anti_hack_firmware", "Durcit les drones contre les intrusions, brouillages et prises de controle ennemies.", "Un essaim pirate n'est plus une armee, c'est une catastrophe."),
		researchSeed("research_drone_carrier_protocol", "Coordonne les drones avec des plateformes porteuses pour soutenir des operations plus longues.", "La portee d'un essaim se mesure a son nid."),
		researchSeed("research_nexus_swarm", "Transforme les drones en reseau Nexus, capable d'agir vite, loin et avec une autonomie avancee.", "Le ciel devient une pensee collective."),

		researchSeed("research_agent_basics", "Pose les bases des agents IA utiles a la cite: observation, proposition et execution controlee.", "La premiere IA utile est celle qui sait quand se taire."),
		researchSeed("research_prompt_memory", "Donne aux agents une memoire de consignes pour rendre leurs actions plus coherentes dans le temps.", "Une instruction oubliee coute parfois plus cher qu'une erreur."),
		researchSeed("research_multi_agent_routing", "Permet a plusieurs agents IA de se partager les taches sans se bloquer mutuellement.", "Quand les agents se parlent, le conseil respire."),
		researchSeed("research_byoai_cost_control", "Controle les couts des modeles IA externes et evite que l'automatisation devienne une fuite de credits.", "L'intelligence la plus chere n'est pas toujours la plus rentable."),
		researchSeed("research_local_llm_optimization", "Optimise les modeles locaux pour garder des decisions IA rapides, privees et moins couteuses.", "Une pensee locale reste sous ton toit."),
		researchSeed("research_autonomous_proposal_system", "Autorise les IA a proposer des plans complets avant validation humaine ou politique.", "Le conseiller devient stratege, pas souverain."),
		researchSeed("research_nexus_cognitive_mesh", "Relie les IA majeures en maillage cognitif pour coordonner ville, guerre, diplomatie et lore.", "La cite commence a rever avec plusieurs voix."),

		researchSeed("research_emissary_protocol", "Structure les premiers contacts diplomatiques et rend les envoyes plus efficaces avec les factions.", "Un protocole evite parfois dix excuses."),
		researchSeed("research_faction_language_models", "Modele les langages, tabous et priorites de faction pour negocier sans provoquer d'incident inutile.", "Traduire les mots ne suffit pas; il faut traduire les interets."),
		researchSeed("research_reputation_mapping", "Cartographie la reputation de la cite et revele les leviers qui influencent allies, rivaux et neutres.", "La reputation est une ressource qui refuse les coffres."),
		researchSeed("research_treaty_simulation", "Simule les traites avant signature pour prevoir gains, risques et trahisons probables.", "Un mauvais accord se voit mieux avant l'encre."),
		researchSeed("research_conflict_mediation", "Ouvre des options de mediation pour calmer des tensions sans deployer l'armee.", "La paix est parfois une manoeuvre de haute precision."),
		researchSeed("research_tribunal_diplomacy", "Relie diplomatie et tribunal pour appuyer les decisions politiques par des preuves et arbitrages.", "La loi donne du poids aux promesses."),
		researchSeed("research_nexus_concordat", "Debloque une doctrine de concordat Nexus, capable de federer plusieurs puissances autour de la cite.", "A ce niveau, signer un traite revient a dessiner une frontiere."),

		researchSeed("research_regional_scanning", "Scanne les regions proches pour reperer menaces, ressources et evenements avant qu'ils arrivent.", "Voir plus loin, c'est deja agir plus tot."),
		researchSeed("research_event_forecasting", "Prevoit les evenements monde avec plus d'avance pour preparer reponses, stocks et forces.", "L'avenir ne parle pas fort, mais il laisse des traces."),
		researchSeed("research_conflict_mapping", "Cartographie les zones de conflit et les acteurs impliques pour choisir ou intervenir.", "Une guerre mal lue devient toujours plus grande."),
		researchSeed("research_weather_adaptation", "Adapte la cite et ses operations aux conditions meteo extremes et anomalies regionales.", "La pluie n'est pas un obstacle quand elle est deja dans le plan."),
		researchSeed("research_world_action_slots", "Ajoute des capacites d'action monde pour intervenir sur plusieurs crises ou opportunites.", "Plus de mains sur la carte, moins de crises ignorees."),
		researchSeed("research_continental_strategy", "Eleve la lecture strategique au niveau continental et augmente l'influence hors des murs.", "La cite apprend a penser en lignes de front."),
		researchSeed("research_nexus_world_awareness", "Donne une conscience monde complete, reliant evenements, factions, risques et opportunites Nexus.", "Le monde n'est plus un decor; il repond."),

		researchSeed("research_guild_charter", "Redige la charte qui rend les actions de guilde possibles et lisibles pour tous les membres.", "Une guilde sans charte est une foule avec un logo."),
		researchSeed("research_cooperative_missions", "Ouvre des missions cooperatives ou plusieurs joueurs peuvent contribuer a un meme objectif.", "La recompense est meilleure quand le risque est partage."),
		researchSeed("research_donation_efficiency", "Ameliore les dons de guilde pour que chaque ressource envoyee profite davantage au collectif.", "Donner juste vaut mieux que donner beaucoup."),
		researchSeed("research_guild_ai_strategy", "Ajoute une strategie IA de guilde pour recommander objectifs, roles et priorites communes.", "Le conseil gagne une voix qui ne cherche pas de titre."),
		researchSeed("research_shared_logistics", "Met en commun une partie de la logistique pour soutenir les membres en crise.", "Une route partagee peut sauver une cite isolee."),
		researchSeed("research_alliance_protocols", "Formalise alliances, renforts et responsabilites entre groupes de joueurs.", "Une alliance solide sait deja quoi faire le jour ou tout casse."),
		researchSeed("research_nexus_guild_network", "Transforme la guilde en reseau Nexus capable d'influencer economie, guerre et monde.", "La banniere devient infrastructure."),

		researchSeed("research_archive_indexing", "Indexe les archives pour retrouver rapidement evenements, preuves, personnages et decisions passees.", "La memoire inutile est celle qu'on ne sait pas relire."),
		researchSeed("research_rumor_tracking", "Suit les rumeurs et signaux faibles pour detecter quetes, conflits et opportunites cachees.", "Chaque murmure cherche le bon archiviste."),
		researchSeed("research_event_canonization", "Transforme certains evenements en canon officiel du monde Nexus et stabilise leurs consequences.", "Ce qui est inscrit commence a compter."),
		researchSeed("research_ai_summary_engine", "Resume les grands evenements avec l'aide de l'IA pour rendre le lore utile aux decisions.", "Un bon resume peut eviter une mauvaise guerre."),
		researchSeed("research_regional_memory", "Donne une memoire propre aux regions afin que leurs histoires influencent les evenements futurs.", "Une region qui se souvient devient un personnage."),
		researchSeed("research_tribunal_archives", "Relie archives et tribunal pour conserver preuves, verdicts et precedents consultables.", "Le passe devient recevable."),
		researchSeed("research_living_lore_engine", "Active un moteur de lore vivant ou actions, quetes et monde s'alimentent mutuellement.", "L'histoire cesse d'etre ecrite apres coup."),

		researchSeed("research_evidence_handling", "Standardise la collecte des preuves pour que les conflits puissent etre juges proprement.", "Une preuve mal gardee devient une rumeur chere."),
		researchSeed("research_witness_protocol", "Protege et organise les temoignages pour renforcer la credibilite des decisions du tribunal.", "Un temoin entendu vaut mieux qu'une foule en colere."),
		researchSeed("research_jury_simulation", "Simule les reactions d'un jury pour anticiper verdicts, biais et risques politiques.", "La justice aussi gagne a tester ses angles morts."),
		researchSeed("research_verdict_impact", "Mesure l'impact des verdicts sur factions, guildes, morale et reputation.", "Un jugement finit rarement dans la salle ou il est prononce."),
		researchSeed("research_guild_arbitration", "Ouvre l'arbitrage de guilde pour resoudre conflits internes et litiges entre alliances.", "Mieux vaut un arbitrage dur qu'une guerre de bannieres."),
		researchSeed("research_faction_law", "Integre les droits de faction pour rendre les verdicts compatibles avec plusieurs cultures politiques.", "La loi commune commence par les differences qu'elle accepte."),
		researchSeed("research_nexus_tribunal_authority", "Donne au tribunal une autorite Nexus capable d'imposer ses decisions sur les grands conflits.", "Quand le verdict tombe, meme les puissants calculent leur silence."),
	}
}

func researchSeed(contentID, description, flavor string) contentDescriptionSeed {
	return contentDescriptionSeed{
		Domain:      "research",
		ContentID:   contentID,
		Description: description,
		Flavor:      flavor,
	}
}

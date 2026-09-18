-- Populate the toggle-style product catalogs (Digital Wall Station, Mobile
-- Digital Station, Mobile Event Pod) with the configuration/enhancement data
-- the Funding OS solutions step renders. Keys match what the wizard reads:
-- baseProjectCost (dollars), traumaConfiguration, optionalEnhancements.

UPDATE products SET catalog = catalog || '{
  "baseProjectCost": 22490,
  "traumaConfiguration": {
    "default": "basic",
    "options": [
      {
        "id": "basic",
        "label": "ACS Basic Bleeding Control Kits",
        "priceAdd": 0,
        "includes": [
          "8 ACS Basic Bleeding Control Kits",
          "2 ACS Bleeding Control Belts",
          "Tourniquets and trauma dressings",
          "Cabinet integration"
        ]
      },
      {
        "id": "advanced",
        "label": "ACS Advanced Bleeding Control Kits",
        "priceAdd": 400,
        "includes": [
          "8 ACS Advanced Bleeding Control Kits",
          "QuikClot hemostatic dressings",
          "HyFin Chest Seal Twin Pack",
          "Survival blankets"
        ]
      }
    ]
  },
  "optionalEnhancements": [
    {
      "id": "security-awareness",
      "label": "Security & Situational Awareness Package",
      "priceAdd": 500,
      "note": "Door/window status, motion, and environmental sensors with alert escalation protocols and visual dashboards",
      "includes": [
        "Door/window status sensors",
        "Motion and environmental sensors",
        "Alert escalation protocols",
        "Visual dashboards"
      ]
    },
    {
      "id": "temp-storage",
      "label": "Temperature-Controlled Storage Module",
      "priceAdd": 850,
      "note": "Climate-controlled medication storage for temperature-sensitive supplies, vaccines, and emergency medications",
      "includes": [
        "Climate-controlled storage compartment",
        "Temperature monitoring",
        "Vaccine and medication safe"
      ]
    }
  ]
}'::jsonb
WHERE slug = 'acs-digital-wall-station';

UPDATE products SET catalog = catalog || '{
  "baseProjectCost": 25500,
  "traumaConfiguration": {
    "default": "basic",
    "options": [
      {
        "id": "basic",
        "label": "ACS Basic Bleeding Control Kits",
        "priceAdd": 0,
        "includes": [
          "8 ACS Basic Bleeding Control Kits",
          "2 ACS Bleeding Control Belts",
          "Tourniquets and trauma dressings",
          "Cabinet integration"
        ]
      },
      {
        "id": "advanced",
        "label": "ACS Advanced Bleeding Control Kits",
        "priceAdd": 400,
        "includes": [
          "8 ACS Advanced Bleeding Control Kits",
          "QuikClot hemostatic dressings",
          "HyFin Chest Seal Twin Pack",
          "Survival blankets"
        ]
      }
    ]
  },
  "optionalEnhancements": [
    {
      "id": "security-awareness",
      "label": "Security & Situational Awareness Package",
      "priceAdd": 500,
      "note": "Door/window status, motion, and environmental sensors with alert escalation protocols and visual dashboards",
      "includes": [
        "Door/window status sensors",
        "Motion and environmental sensors",
        "Alert escalation protocols",
        "Visual dashboards"
      ]
    },
    {
      "id": "temp-storage",
      "label": "Temperature-Controlled Storage Module",
      "priceAdd": 850,
      "note": "Climate-controlled medication storage for temperature-sensitive supplies, vaccines, and emergency medications",
      "includes": [
        "Climate-controlled storage compartment",
        "Temperature monitoring",
        "Vaccine and medication safe"
      ]
    }
  ]
}'::jsonb
WHERE slug = 'acs-mobile-digital-station';

UPDATE products SET catalog = catalog || '{
  "baseProjectCost": 18600,
  "traumaConfiguration": {
    "default": "basic",
    "options": [
      {
        "id": "basic",
        "label": "ACS Basic Bleeding Control Kits",
        "priceAdd": 0,
        "includes": [
          "8 ACS Basic Bleeding Control Kits",
          "2 ACS Bleeding Control Belts",
          "Tourniquets and trauma dressings",
          "Cabinet integration"
        ]
      },
      {
        "id": "advanced",
        "label": "ACS Advanced Bleeding Control Kits",
        "priceAdd": 400,
        "includes": [
          "8 ACS Advanced Bleeding Control Kits",
          "QuikClot hemostatic dressings",
          "HyFin Chest Seal Twin Pack",
          "Survival blankets"
        ]
      }
    ]
  },
  "optionalEnhancements": [
    {
      "id": "security-awareness",
      "label": "Security & Situational Awareness Package",
      "priceAdd": 500,
      "note": "Door/window status, motion, and environmental sensors with alert escalation protocols and visual dashboards",
      "includes": [
        "Door/window status sensors",
        "Motion and environmental sensors",
        "Alert escalation protocols",
        "Visual dashboards"
      ]
    },
    {
      "id": "temp-storage",
      "label": "Temperature-Controlled Storage Module",
      "priceAdd": 850,
      "note": "Climate-controlled medication storage for temperature-sensitive supplies, vaccines, and emergency medications",
      "includes": [
        "Climate-controlled storage compartment",
        "Temperature monitoring",
        "Vaccine and medication safe"
      ]
    }
  ]
}'::jsonb
WHERE slug = 'acs-mobile-event-pod';

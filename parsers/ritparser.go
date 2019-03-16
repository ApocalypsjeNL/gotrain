package parsers

import (
	"io"

	"github.com/beevik/etree"
	"github.com/rijdendetreinen/gotrain/models"
)

// ParseRitMessage parses a RIT XML message to a Service object
func ParseRitMessage(reader io.Reader) models.Service {
	doc := etree.NewDocument()

	if _, err := doc.ReadFrom(reader); err != nil {
		panic(err)
	}

	product := doc.SelectElement("PutReisInformatieBoodschapIn").SelectElement("ReisInformatieProductRitInfo")
	productAdministration := product.SelectElement("RIPAdministratie")
	infoProduct := product.SelectElement("RitInfo")

	var service models.Service

	service.Timestamp = productAdministration.SelectElement("ReisInformatieTijdstip").Text()
	service.ProductID = productAdministration.SelectElement("ReisInformatieProductID").Text()
	service.ValidUntil = productAdministration.SelectElement("GeldigTot").Text()

	service.ID = infoProduct.SelectElement("TreinNummer").Text()
	service.ServiceDate = infoProduct.SelectElement("TreinDatum").Text()

	service.ServiceNumber = infoProduct.SelectElement("TreinNummer").Text()
	service.ServiceType = infoProduct.SelectElement("TreinSoort").Text()
	service.ServiceTypeCode = infoProduct.SelectElement("TreinSoort").SelectAttrValue("Code", "")
	service.Company = infoProduct.SelectElement("Vervoerder").Text()

	service.ReservationRequired = ParseInfoPlusBoolean(infoProduct.SelectElement("TreinNummer"))
	service.WithSupplement = ParseInfoPlusBoolean(infoProduct.SelectElement("Toeslag"))
	service.SpecialTicket = ParseInfoPlusBoolean(infoProduct.SelectElement("SpeciaalKaartje"))
	service.JourneyPlanner = ParseInfoPlusBoolean(infoProduct.SelectElement("Reisplanner"))

	service.Modifications = ParseInfoPlusModifications(infoProduct.SelectElement("LogischeRit"))

	for _, partInfo := range infoProduct.SelectElement("LogischeRit").SelectElements("LogischeRitDeel") {
		var servicePart models.ServicePart

		servicePart.ServiceNumber = partInfo.SelectElement("LogischeRitDeelNummer").Text()
		servicePart.Modifications = ParseInfoPlusModifications(partInfo)

		for _, stopInfo := range partInfo.SelectElements("LogischeRitDeelStation") {
			var serviceStop models.ServiceStop

			serviceStop.Station = ParseInfoPlusStation(stopInfo.SelectElement("Station"))
			serviceStop.Modifications = ParseInfoPlusModifications(infoProduct)

			// Stop behavior:
			serviceStop.StopType = ParseOptionalText(stopInfo.SelectElement("StationnementType"))
			serviceStop.DoNotBoard = ParseInfoPlusBoolean(stopInfo.SelectElement("NietInstappen"))

			if stopInfo.SelectElement("Stopt") != nil {
				serviceStop.StoppingActual = ParseInfoPlusBoolean(ParseWhenAttribute(stopInfo, "Stopt", "InfoStatus", "Actueel"))
				serviceStop.StoppingPlanned = ParseInfoPlusBoolean(ParseWhenAttribute(stopInfo, "Stopt", "InfoStatus", "Gepland"))
			}

			// Arrival/departure time:
			if stopInfo.SelectElement("AankomstTijd") != nil {
				serviceStop.ArrivalTime = ParseInfoPlusDateTime(ParseWhenAttribute(stopInfo, "AankomstTijd", "InfoStatus", "Gepland"))
			}
			if stopInfo.SelectElement("VertrekTijd") != nil {
				serviceStop.DepartureTime = ParseInfoPlusDateTime(ParseWhenAttribute(stopInfo, "VertrekTijd", "InfoStatus", "Gepland"))
			}

			// Platforms
			if stopInfo.SelectElement("TreinAankomstSpoor") != nil {
				serviceStop.ArrivalPlatformActual = ParseInfoPlusPlatform(ParseWhenAttributeMulti(stopInfo, "TreinAankomstSpoor", "InfoStatus", "Actueel"))
				serviceStop.ArrivalPlatformPlanned = ParseInfoPlusPlatform(ParseWhenAttributeMulti(stopInfo, "TreinAankomstSpoor", "InfoStatus", "Gepland"))
			}
			if stopInfo.SelectElement("TreinVertrekSpoor") != nil {
				serviceStop.DeparturePlatformActual = ParseInfoPlusPlatform(ParseWhenAttributeMulti(stopInfo, "TreinVertrekSpoor", "InfoStatus", "Actueel"))
				serviceStop.DeparturePlatformPlanned = ParseInfoPlusPlatform(ParseWhenAttributeMulti(stopInfo, "TreinVertrekSpoor", "InfoStatus", "Gepland"))
			}

			// Delays
			if stopInfo.SelectElement("ExacteAankomstVertraging") != nil {
				serviceStop.ArrivalDelay = ParseInfoPlusDuration(stopInfo.SelectElement("ExacteAankomstVertraging"))
			}
			if stopInfo.SelectElement("ExacteVertrekVertraging") != nil {
				serviceStop.DepartureDelay = ParseInfoPlusDuration(stopInfo.SelectElement("ExacteVertrekVertraging"))
			}

			for _, materialInfo := range stopInfo.SelectElements("MaterieelDeel") {
				var material models.Material

				material.NaterialType = materialInfo.SelectElement("MaterieelDeelSoort").Text() + "-" + materialInfo.SelectElement("MaterieelDeelAanduiding").Text()
				material.Accessible = ParseInfoPlusBoolean(materialInfo.SelectElement("MaterieelDeelToegankelijk"))
				material.RemainsBehind = ParseInfoPlusBoolean(materialInfo.SelectElement("AchterBlijvenMaterieelDeel"))

				if materialInfo.SelectElement("MaterieelNummer") != nil {
					material.Number = materialInfo.SelectElement("MaterieelNummer").Text()
				}

				material.DestinationActual = ParseInfoPlusStation(ParseWhenAttribute(materialInfo, "MaterieelDeelEindBestemming", "InfoStatus", "Actueel"))
				material.DestinationPlanned = ParseInfoPlusStation(ParseWhenAttribute(materialInfo, "MaterieelDeelEindBestemming", "InfoStatus", "Gepland"))

				serviceStop.Material = append(serviceStop.Material, material)
			}

			servicePart.Stops = append(servicePart.Stops, serviceStop)
		}

		service.ServiceParts = append(service.ServiceParts, servicePart)
	}

	return service
}
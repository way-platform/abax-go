package abax

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/way-platform/abax-go/internal/oapi/abaxoapi"
	abaxv1 "github.com/way-platform/abax-go/proto/gen/go/wayplatform/connect/abax/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ListVehiclesRequest is the request for the [Client.ListVehicles] method.
type ListVehiclesRequest struct {
	Page     int32 `json:"page,omitempty"`
	PageSize int32 `json:"pageSize,omitempty"`
}

// ListVehiclesResponse is the response for the [Client.ListVehicles] method.
type ListVehiclesResponse struct {
	Page     int32             `json:"page"`
	PageSize int32             `json:"pageSize"`
	Vehicles []*abaxv1.Vehicle `json:"items"`
}

// ListVehicles lists all vehicles in the current organization.
//
// Required scopes:
// - abax_profile
// - open_api
// - open_api.vehicles
func (c *Client) ListVehicles(ctx context.Context, request *ListVehiclesRequest) (*ListVehiclesResponse, error) {
	httpRequest, err := c.newRequest(ctx, "GET", "/v1/vehicles", nil)
	if err != nil {
		return nil, err
	}
	query := httpRequest.URL.Query()
	query.Add("page", strconv.Itoa(int(request.Page)))
	query.Add("page_size", strconv.Itoa(int(request.PageSize)))
	httpRequest.URL.RawQuery = query.Encode()
	httpResponse, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer httpResponse.Body.Close() //nolint:errcheck
	if httpResponse.StatusCode != http.StatusOK {
		return nil, c.newResponseError(httpResponse)
	}
	body, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, err
	}
	var rawResponse struct {
		Page     int32               `json:"page"`
		PageSize int32               `json:"page_size"`
		Items    []*abaxoapi.Vehicle `json:"items"`
	}
	if err := json.Unmarshal(body, &rawResponse); err != nil {
		return nil, err
	}
	response := ListVehiclesResponse{
		Vehicles: make([]*abaxv1.Vehicle, 0, len(rawResponse.Items)),
		Page:     rawResponse.Page,
		PageSize: rawResponse.PageSize,
	}
	for _, item := range rawResponse.Items {
		vehicle, err := vehicleToProto(item)
		if err != nil {
			return nil, err
		}
		response.Vehicles = append(response.Vehicles, vehicle)
	}
	return &response, nil
}

func vehicleToProto(input *abaxoapi.Vehicle) (*abaxv1.Vehicle, error) {
	var output abaxv1.Vehicle

	// Basic string fields
	if input.ID != nil {
		output.SetId(*input.ID)
	}
	if input.AssetID != nil {
		output.SetAssetId(*input.AssetID)
	}
	if input.VIN != nil {
		output.SetVin(*input.VIN)
	}
	if input.Alias != nil {
		output.SetAlias(*input.Alias)
	}

	// Manufacturer and model
	if input.Manufacturer != nil && input.Manufacturer.Name != nil {
		output.SetManufacturerName(*input.Manufacturer.Name)
	}
	if input.Model != nil && input.Model.Name != nil {
		output.SetModelName(*input.Model.Name)
	}

	// License plate and registration time
	if input.LicensePlate != nil {
		if input.LicensePlate.Number != nil {
			output.SetLicensePlateNumber(*input.LicensePlate.Number)
		}
		if input.LicensePlate.RegistrationDate != nil {
			output.SetLicensePlateRegistrationTime(timestamppb.New(*input.LicensePlate.RegistrationDate))
		}
	}

	// Commercial class with unknown handling
	if input.CommercialClass != nil {
		switch *input.CommercialClass {
		case abaxoapi.VehicleCommercialClassPrivate:
			output.SetCommercialClass(abaxv1.Vehicle_PRIVATE)
		case abaxoapi.VehicleCommercialClassCompany:
			output.SetCommercialClass(abaxv1.Vehicle_COMPANY)
		case abaxoapi.VehicleCommercialClassCommercial:
			output.SetCommercialClass(abaxv1.Vehicle_COMMERCIAL)
		case abaxoapi.VehicleCommercialClassCommercialWithPrivateUse:
			output.SetCommercialClass(abaxv1.Vehicle_COMMERCIAL_WITH_PRIVATE_USE)
		case abaxoapi.VehicleCommercialClassUnknown:
			output.SetCommercialClass(abaxv1.Vehicle_COMMERCIAL_CLASS_NOT_AVAILABLE)
		default:
			output.SetCommercialClass(abaxv1.Vehicle_COMMERCIAL_CLASS_UNKNOWN)
			output.SetUnknownCommercialClass(string(*input.CommercialClass))
		}
	}

	// Registered time
	if input.RegisteredAt != nil {
		output.SetRegisteredTime(timestamppb.New(*input.RegisteredAt))
	}

	// Unit fields
	if input.Unit != nil {
		if input.Unit.ID != nil {
			output.SetUnitId(*input.Unit.ID)
		}
		if input.Unit.SerialNumber != nil {
			output.SetUnitSerialNumber(*input.Unit.SerialNumber)
		}
		if input.Unit.Type != nil {
			output.SetUnitType(*input.Unit.Type)
		}

		// Unit health
		if input.Unit.Health != nil {
			switch *input.Unit.Health {
			case abaxoapi.UnitHealthHealthy:
				output.SetUnitHealth(abaxv1.Vehicle_HEALTHY)
			case abaxoapi.UnitHealthDegraded:
				output.SetUnitHealth(abaxv1.Vehicle_DEGRADED)
			case abaxoapi.UnitHealthUnhealthy:
				output.SetUnitHealth(abaxv1.Vehicle_UNHEALTHY)
			case abaxoapi.UnitHealthUnknown:
				output.SetUnitHealth(abaxv1.Vehicle_UNIT_HEALTH_NOT_AVAILABLE)
			default:
				output.SetUnitHealth(abaxv1.Vehicle_UNIT_HEALTH_UNKNOWN)
				output.SetUnknownUnitHealth(string(*input.Unit.Health))
			}
		}

		// Unit status
		if input.Unit.Status != nil {
			switch *input.Unit.Status {
			case abaxoapi.UnitStatusActive:
				output.SetUnitStatus(abaxv1.Vehicle_ACTIVE)
			case abaxoapi.UnitStatusDeactivated:
				output.SetUnitStatus(abaxv1.Vehicle_DEACTIVATED)
			case abaxoapi.UnitStatusUnknown:
				output.SetUnitStatus(abaxv1.Vehicle_UNIT_STATUS_NOT_AVAILABLE)
			default:
				output.SetUnitStatus(abaxv1.Vehicle_UNIT_STATUS_UNKNOWN)
				output.SetUnknownUnitStatus(string(*input.Unit.Status))
			}
		}
	}

	// Location
	if input.Location != nil {
		location, err := locationToProto(input.Location)
		if err != nil {
			return nil, err
		}
		output.SetLocation(location)
	}

	// Driver fields
	if input.Driver != nil {
		if input.Driver.ID != nil {
			output.SetDriverId(*input.Driver.ID)
		}
		if input.Driver.ExternalID != nil {
			output.SetDriverExternalId(*input.Driver.ExternalID)
		}
		if input.Driver.Name != nil {
			output.SetDriverName(*input.Driver.Name)
		}
	}

	// Organization
	if input.Organization != nil {
		if input.Organization.ID != nil {
			output.SetOrganizationId(*input.Organization.ID)
		}
		if input.Organization.Name != nil {
			output.SetOrganizationName(*input.Organization.Name)
		}
	}

	// Odometer
	if input.Odometer != nil {
		if input.Odometer.Value != nil {
			output.SetOdometerM(float64(*input.Odometer.Value))
		}
		if input.Odometer.Timestamp != nil {
			output.SetOdometerTime(timestamppb.New(*input.Odometer.Timestamp))
		}
	}

	// Notes
	if input.Notes != nil {
		output.SetNotes(*input.Notes)
	}

	// Fuel type
	if input.FuelType != nil {
		switch *input.FuelType {
		case abaxoapi.FuelTypePetrol:
			output.SetFuelType(abaxv1.Vehicle_PETROL)
		case abaxoapi.FuelTypeElectricity:
			output.SetFuelType(abaxv1.Vehicle_ELECTRICITY)
		case abaxoapi.FuelTypeDiesel:
			output.SetFuelType(abaxv1.Vehicle_DIESEL)
		case abaxoapi.FuelTypeLpg:
			output.SetFuelType(abaxv1.Vehicle_LPG)
		case abaxoapi.FuelTypeDieselHybrid:
			output.SetFuelType(abaxv1.Vehicle_DIESEL_HYBRID)
		case abaxoapi.FuelTypePetrolHybrid:
			output.SetFuelType(abaxv1.Vehicle_PETROL_HYBRID)
		case abaxoapi.FuelTypeUnknown:
			output.SetFuelType(abaxv1.Vehicle_FUEL_TYPE_NOT_AVAILABLE)
		default:
			output.SetFuelType(abaxv1.Vehicle_FUEL_TYPE_UNKNOWN)
			output.SetUnknownFuelType(string(*input.FuelType))
		}
	}

	// Fuel consumption
	if input.FuelConsumption != nil {
		output.SetFuelConsumptionVaries(float64(*input.FuelConsumption))
	}

	// Engine size
	if input.EngineSize != nil {
		output.SetEngineSizeCc(float64(*input.EngineSize))
	}

	// Color
	if input.Color != nil {
		output.SetColor(*input.Color)
	}

	// CO2 emissions
	if input.Co2Emissions != nil {
		output.SetCo2EmissionsGKm(float64(*input.Co2Emissions))
	}

	return &output, nil
}

func locationToProto(input *abaxoapi.Location) (*abaxv1.Location, error) {
	var output abaxv1.Location

	// Latitude and longitude
	if input.Latitude != nil {
		output.SetLatitude(float64(*input.Latitude))
	}
	if input.Longitude != nil {
		output.SetLongitude(float64(*input.Longitude))
	}

	// Speed
	if input.Speed != nil {
		output.SetSpeedKmh(float64(*input.Speed))
	}

	// In movement
	if input.InMovement != nil {
		output.SetInMovement(*input.InMovement)
	}

	// Course
	if input.Course != nil {
		output.SetCourseDeg(float64(*input.Course))
	}

	// Timestamp
	if input.Timestamp != nil {
		output.SetTimestamp(timestamppb.New(*input.Timestamp))
	}

	// Signal source
	if input.SignalSource != nil {
		switch *input.SignalSource {
		case abaxoapi.SignalSourceGps:
			output.SetSignalSource(abaxv1.Location_GPS)
		case abaxoapi.SignalSourceGsm:
			output.SetSignalSource(abaxv1.Location_GSM)
		default:
			output.SetSignalSource(abaxv1.Location_SIGNAL_SOURCE_UNKNOWN)
			output.SetUnknownSignalSource(string(*input.SignalSource))
		}
	}

	// Accuracy radius
	if input.AccuracyRadius != nil {
		output.SetAccuracyRadiusM(float64(*input.AccuracyRadius))
	}

	return &output, nil
}

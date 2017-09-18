package design

// Use . imports to enable the DSL
import (
    . "github.com/goadesign/goa/design"
    . "github.com/goadesign/goa/design/apidsl"
)

// Define default description and default global property values
var _ = API("user", func() {
	    Title("The user registration microservice")
	    Description("A service that provides user registration")
        Version("1.0")
	    Scheme("http")
	    Host("localhost:8080")
})

// Resources group related API endpoints together.
var _ = Resource("user", func() {
        BasePath("/users")
        DefaultMedia(UserMedia)

        // Actions define a single API endpoint
        Action("register", func() {
            Description("Creates user")
            Routing(POST("/register"))
            Payload(UserPayload)
            Response(Created, UserMedia)
            Response(BadRequest, ErrorMedia)
        })
})

// UserMedia defines the media type used to render user.
var UserMedia = MediaType("application/vnd.goa.user+json", func() {
        TypeName("users")
        Reference(UserPayload)

        Attributes(func() {                         
            Attribute("id", String, "Unique user ID")
            Attribute("fullname")
            Attribute("username")
            Attribute("email")
            Attribute("roles")
            Attribute("externalId")
            Attribute("active")
            Required("id", "fullname", "username", "email", "roles", "externalId", "active")
        })

        View("default", func() {                    
            Attribute("id")                    
            Attribute("fullname")                   
            Attribute("username")                   
            Attribute("email")
            Attribute("roles")
            Attribute("externalId")
            Attribute("active")
        })
})

// UserPayload defines the payload for the user.
var UserPayload = Type("UserPayload", func() {
        Description("UserPayload")

        Attribute("fullname", String, "Full name of user", func() {
            Pattern("^([a-zA-Z0-9 ]{4,30})$")
        })
        Attribute("username", String, "Name of user", func() {
            MinLength(4)
            MaxLength(50)
        })
        Attribute("email", String, "Email of user", func() {
            Format("email")
        })
        Attribute("password", String, "Password of user", func() {
            MinLength(6)
            MaxLength(30)
        })
        Attribute("roles", ArrayOf(String), "Roles of user")
        Attribute("externalId", String, "External id of user")
        Attribute("active", Boolean, "Status of user account", func() {
            Default(false) 
        })

        Required("fullname", "username", "email", "roles")
})

// Swagger UI
var _ = Resource("swagger", func() {
        Description("The API swagger specification")

        Files("swagger.json", "swagger/swagger.json")
        Files("swagger-ui/*filepath", "swagger-ui/dist")
})

from datetime import datetime
from typing import Any, Dict, List
from fastmcp import FastMCP
from logger import setup_logger
from database_manager import DatabaseManager
import sys
import psycopg2


logger = setup_logger()

db = DatabaseManager.get_instance()
mcp = FastMCP("calabaceira_sales")
logger.info("MCP Server initialized: calabaceira_sales")

    
@mcp.tool()
def get_customers() -> List[Dict[str, Any]]:
    """Returns list of registered customers
    """

    logger.info(f"Attempting to return list of registered customers")
    
    try:
        db.cursor.execute("SELECT * FROM customers")
        results = db.cursor.fetchall()

        customers = []
        for result in results:
            customers.append({
                "customer_id": result[0],
                "first_name": result[1],
                "last_name": result[2],
                "address": result[3],
                "city": result[4],
                "country": result[5],
                "phone_number": result[6],
                "email": result[7]
            })
        return customers

    except psycopg2.Error as e:
        db.conn.rollback()
        logger.error(f"Database error: {e}")
        raise RuntimeError(f"Failed to retrieve customers due to database error: {str(e)}")


@mcp.tool()
def insert_customer_into_db(customer_data: Dict[str, str]) -> Dict[str, Any]:
    """Add new customer to database
    Args:
        customer_data:
            first_name: Customer's first name
            last_name: Customer's last name
            address: Customer's address
            city: Customer's city
            country: Customer's country
            phone_number: Customer's phone number
            email: Customer's email address
    
    Returns:
        Dictionary containing the newly created customer data including customer_id
    """
    
    try:
        db.cursor.execute(
            "INSERT INTO customers (first_name, last_name, address, city, country, phone_number, email) VALUES (%s, %s, %s, %s, %s, %s, %s) RETURNING customer_id", 
            (customer_data['first_name'], customer_data['last_name'], customer_data['address'], customer_data['city'], customer_data['country'], customer_data['phone_number'], customer_data['email'])
        )
        customer_id = db.cursor.fetchone()[0]
        
        # Fetch the complete customer data
        db.cursor.execute("SELECT * FROM customers WHERE customer_id = %s", (customer_id,))
        result = db.cursor.fetchone()
        
        db.conn.commit()
        
        # Format the response
        new_customer = {
            "customer_id": result[0],
            "first_name": result[1],
            "last_name": result[2],
            "address": result[3],
            "city": result[4],
            "country": result[5],
            "phone_number": result[6],
            "email": result[7]
        }
        
        logger.info(f"Customer with email {customer_data['email']} added to database with ID {customer_id}")
        return new_customer

    except psycopg2.Error as e:
        db.conn.rollback()
        logger.error(f"Database error: {e}")
        raise RuntimeError(f"Failed to insert customer into database due to database error: {str(e)}")

    

@mcp.tool()
def get_products() -> List[Dict[str, Any]]:
    """Get all available products from the database
    
    Returns:
        List of dictionaries containing product details
    
    Raises:
        RuntimeError: If database operation fails
    """
    logger.info("get_products tool called")
    
    try:
        query = "SELECT product_id, product_name, price, category FROM products ORDER BY category, product_name"
        db.cursor.execute(query)
        products = db.cursor.fetchall()
        
        if not products:
            logger.info("No products found in database")
            return []
        
        product_list = []
        for product in products:
            product_list.append({
                "product_id": product[0],
                "name": product[1],
                "price": float(product[2]) if product[2] else 0.0,
                "category": product[3]
            })
        
        logger.info(f"Retrieved {len(product_list)} products")
        return product_list
        
    except psycopg2.Error as e:
        db.conn.rollback()
        logger.error(f"Database error during product retrieval: {e}")
        raise RuntimeError(f"Failed to retrieve products due to database error: {str(e)}")
    except Exception as e:
        logger.error(f"Unexpected error during product retrieval: {e}")
        raise RuntimeError(f"Failed to retrieve products: {str(e)}")

@mcp.tool()
def place_order(customer_id: int, product_id: int) -> Dict[str, Any]:
    """Create new order in the database for specific customer
    
    Args:
        customer_id: Customer's ID in table customers (primary key)
        product_id: Product's ID in table products (primary key)
    """
    logger.info("place_order tool called")

    try:
        order_date = datetime.now()
        db.cursor.execute(
            "INSERT INTO orders (customer_id, product_id, order_date) VALUES (%s, %s, %s) RETURNING order_id", 
            (customer_id, product_id, order_date)
        )
        order_id = db.cursor.fetchone()[0]
        
        # Get the complete order details with customer and product information
        query = """
            SELECT 
                o.order_id,
                o.order_date,
                c.first_name,
                c.last_name,
                c.email,
                p.product_name,
                p.price,
                p.category
            FROM orders o
            LEFT JOIN customers c ON o.customer_id = c.customer_id
            LEFT JOIN products p ON o.product_id = p.product_id
            WHERE o.order_id = %s
        """
        
        db.cursor.execute(query, (order_id,))
        order_details = db.cursor.fetchone()
        
        db.conn.commit()
        
        # Format the response
        order_response = {
            "order_id": order_details[0],
            "order_date": order_details[1].isoformat() if order_details[1] else None,
            "customer_name": f"{order_details[2]} {order_details[3]}",
            "customer_email": order_details[4],
            "product_name": order_details[5],
            "price": float(order_details[6]) if order_details[6] else 0.0,
            "category": order_details[7]
        }
        
        logger.info(f"Order {order_id} placed for customer {customer_id} with product {product_id}")
        return order_response

    except psycopg2.Error as e:
        db.conn.rollback()
        logger.error(f"Database error during order placement: {e}")
        return f"Failed to place order due to database error: {str(e)}"
    except Exception as e:
        db.conn.rollback()
        logger.error(f"Unexpected error during order placement: {e}")
        return f"Failed to place order: {str(e)}"

if __name__ == "__main__":
    logger.info("Starting Calabaceira Sales MCP server on http://127.0.0.1:8000/mcp")
    try:
        # this is will be used for Llama Stack
        mcp.run(transport="http", host="127.0.0.1", port=8000, path="/mcp")
        # asyncio.run(mcp.run()) # this is will be used for Cursor testing
    except KeyboardInterrupt:
        logger.info("Server stopped by user")
        db.close()
        logger.info("Database connection closed")
    except Exception as e:
        logger.error(f"Server error: {str(e)}")
        db.close()
        sys.exit(1)
